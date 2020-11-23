package broadcast

import (
	"container/list"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/sbekti/broadcastd/instagram"
	"golang.org/x/net/websocket"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	cacheTTL          = 60 * time.Second
	maxRecentComments = 100
)

type Broadcast struct {
	streaming    bool
	streamingMux sync.RWMutex

	config  *Config
	server  *Server
	streams map[string]*Stream

	connections    map[*websocket.Conn]struct{}
	connectionsMux sync.RWMutex

	commentsCache  *ttlcache.Cache
	recentComments *list.List
}

func NewBroadcast(c *Config) *Broadcast {
	cache := ttlcache.NewCache()
	cache.SetTTL(cacheTTL)

	b := &Broadcast{
		config:         c,
		streams:        make(map[string]*Stream),
		connections:    make(map[*websocket.Conn]struct{}),
		commentsCache:  cache,
		recentComments: list.New(),
	}
	b.server = NewServer(b, c.BindIP, c.BindPort)

	for name := range c.Accounts {
		b.streams[name] = NewStream(name, b.config, b)
	}

	return b
}

func (b *Broadcast) Start() error {
	return b.server.Start()
}

func (b *Broadcast) Stop() error {
	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		if b.streaming {
			return b.StopStreams()
		}
		return nil
	})

	g.Go(func() error {
		return b.server.Shutdown()
	})

	return g.Wait()
}

func (b *Broadcast) StartStreams() error {
	b.streamingMux.Lock()
	defer b.streamingMux.Unlock()

	if b.streaming {
		return fmt.Errorf("broadcast: streams are already started")
	}

	g, _ := errgroup.WithContext(context.Background())

	for _, stream := range b.streams {
		stream := stream
		g.Go(func() error {
			return stream.Start()
		})
	}

	err := g.Wait()
	if err != nil {
		return err
	}

	b.streaming = true
	return nil
}

func (b *Broadcast) StopStreams() error {
	b.streamingMux.Lock()
	defer b.streamingMux.Unlock()

	if !b.streaming {
		return fmt.Errorf("broadcast: streams are not started")
	}

	g, _ := errgroup.WithContext(context.Background())

	for _, stream := range b.streams {
		stream := stream
		g.Go(func() error {
			return stream.Stop()
		})
	}

	err := g.Wait()
	if err != nil {
		return err
	}

	b.streaming = false
	return nil
}

func (b *Broadcast) broadcastComment(streamName string, broadcastID int, comment instagram.LiveComment) error {
	cacheKey := strconv.FormatInt(comment.PK, 10)
	if _, err := b.commentsCache.Get(cacheKey); err == nil {
		// Comment already exists, skip processing.
		return nil
	}

	if b.config.Logging.Enabled {
		err := b.writeCommentLog(int64(comment.CreatedAt), broadcastID, streamName, comment.User.Username, comment.Text)
		if err != nil {
			return err
		}
	}

	// Mark the comment as seen.
	if err := b.commentsCache.Set(cacheKey, true); err != nil {
		return err
	}

	if b.recentComments.Len() > maxRecentComments {
		b.recentComments.Remove(b.recentComments.Front())
	}
	b.recentComments.PushBack(comment)

	b.connectionsMux.RLock()
	defer b.connectionsMux.RUnlock()

	for c := range b.connections {
		if err := websocket.JSON.Send(c, comment); err != nil {
			return err
		}
	}
	return nil
}

func (b *Broadcast) writeViewerLog(timestamp int64, broadcastID int, username string,
	viewerCount int, totalUniqueViewerCount int) error {

	logDirectory := b.config.Logging.LogDirectory
	if err := os.MkdirAll(logDirectory, os.ModePerm); err != nil {
		return err
	}

	logFileName := fmt.Sprintf("viewers_%s_%d.log", username, broadcastID)
	logFilePath := path.Join(logDirectory, logFileName)
	f, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)
	err = w.Write([]string{
		strconv.FormatInt(timestamp, 10),
		strconv.Itoa(broadcastID),
		username,
		strconv.Itoa(viewerCount),
		strconv.Itoa(totalUniqueViewerCount),
	})
	if err != nil {
		return err
	}

	w.Flush()

	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

func (b *Broadcast) writeCommentLog(timestamp int64, broadcastID int, username string,
	commenter string, comment string) error {

	logDirectory := b.config.Logging.LogDirectory
	if err := os.MkdirAll(filepath.Dir(logDirectory), os.ModePerm); err != nil {
		return err
	}

	logFileName := fmt.Sprintf("comments_%s_%d.log", username, broadcastID)
	logFilePath := path.Join(logDirectory, logFileName)
	f, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)
	err = w.Write([]string{
		strconv.FormatInt(timestamp, 10),
		strconv.Itoa(broadcastID),
		username,
		commenter,
		comment,
	})
	if err != nil {
		return err
	}

	w.Flush()

	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

func (b *Broadcast) writeFinalViewerList(broadcastID int, username string,
	viewerList *instagram.LiveGetFinalViewerListResponse) error {

	logDirectory := b.config.Logging.LogDirectory
	if err := os.MkdirAll(filepath.Dir(logDirectory), os.ModePerm); err != nil {
		return err
	}

	logFileName := fmt.Sprintf("final_viewers_%s_%d.json", username, broadcastID)
	logFilePath := path.Join(logDirectory, logFileName)

	jsonData, err := json.MarshalIndent(viewerList, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(logFilePath, jsonData, 0644); err != nil {
		return err
	}
	return nil
}
