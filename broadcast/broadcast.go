package broadcast

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"sync"
)

type Broadcast struct {
	streaming    bool
	streamingMux sync.Mutex

	config  *Config
	server  *Server
	streams map[string]*Stream
}

func NewBroadcast(c *Config) *Broadcast {
	b := &Broadcast{
		config:  c,
		streams: make(map[string]*Stream),
	}
	b.server = NewServer(b, c.BindIP, c.BindPort)

	for name := range c.Accounts {
		b.streams[name] = NewStream(name, b.config)
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
