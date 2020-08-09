package broadcast

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/sbekti/broadcastd/instagram"
	"golang.org/x/sync/errgroup"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	streamWidth          = 720
	streamHeight         = 1280
	cooldownDelay        = 30 * time.Second
	encoderRestartDelay  = 5 * time.Second
	pollDelay            = 5 * time.Second
	challengeTimeout     = 2 * time.Minute
	jpegQuality          = 95
	numCommentsRequested = 10
)

type Stream struct {
	name          string
	config        *Config
	instagram     *instagram.Instagram
	broadcastID   int
	uploadURL     string
	apiPath       string
	securityCode  chan string
	ctx           context.Context
	cancel        context.CancelFunc
	done          chan error
	startTime     time.Time
	loginRequired bool
	streaming     bool
	streamingMux  sync.Mutex
}

type broadcastStoppedError struct {
	broadcastID int
}

func (e broadcastStoppedError) Error() string {
	return fmt.Sprintf("broadcast %d has stopped", e.broadcastID)
}

type processExitedError struct{}

func (e processExitedError) Error() string {
	return fmt.Sprintf("encoder has exited")
}

func NewStream(name string, config *Config) *Stream {
	var s = &Stream{
		name:          name,
		config:        config,
		instagram:     nil,
		broadcastID:   0,
		uploadURL:     "",
		apiPath:       "",
		securityCode:  make(chan string),
		ctx:           nil,
		cancel:        nil,
		done:          make(chan error, 1),
		startTime:     time.Time{},
		loginRequired: true,
	}

	return s
}

func (s *Stream) Start() error {
	s.streamingMux.Lock()
	defer s.streamingMux.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel

	go s.eventLoop()
	s.streaming = true
	return nil
}

func (s *Stream) Stop() error {
	s.streamingMux.Lock()
	defer s.streamingMux.Unlock()

	if s.streaming {
		s.cancel()
		s.streaming = false
		return <-s.done
	}

	return nil
}

func (s *Stream) eventLoop() {
	for {
		select {
		case <-s.ctx.Done():
			s.done <- nil
			return
		default:
			s.loopCycle()
		}
	}
}

func (s *Stream) loopCycle() {
	if s.loginRequired {
		if err := s.login(); err != nil {
			switch err := err.(type) {
			case *instagram.ChallengeError:
				log.Warnf("stream: %s: challenge code is required", s.name)
				s.apiPath = err.Challenge.APIPath

				if err := s.respondChallenge(); err != nil {
					log.Errorf("stream: %s: unable to complete challenge: %v", s.name, err)
					s.cooldown()
					return
				}
			default:
				log.Errorf("stream: %s: unable to login: %v", s.name, err)
				s.cooldown()
				return
			}
		}

		log.Infof("stream: %s: logged in", s.instagram.Account.Username)
		s.loginRequired = false
	}

	if err := s.createBroadcast(s.config.Notify); err != nil {
		log.Errorf("stream: %s: unable to create broadcast: %v", s.name, err)
		switch err.(type) {
		case *instagram.LoginRequiredError:
			s.loginRequired = true
			return
		default:
			s.cooldown()
			return
		}
	}

	g, ctx := errgroup.WithContext(s.ctx)

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				if err := s.runEncoder(ctx); err != nil {
					log.Errorf("stream: %s: unable to stream broadcast %d: %v", s.name, s.broadcastID, err)
					time.Sleep(encoderRestartDelay)
				}
			}
		}
	})

	g.Go(func() error {
		lastCommentTS := 0

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(pollDelay):
				heartbeat, err := s.heartbeatAndStatus()
				if err != nil {
					return err
				}
				log.Debugf("stream: %s: heartbeat: %+v", s.name, heartbeat)

				newLastCommentTS, err := s.getComments(lastCommentTS)
				if err != nil {
					return err
				}
				lastCommentTS = newLastCommentTS
			}
		}
	})

	if err := g.Wait(); err != nil {
		switch err.(type) {
		case *instagram.LoginRequiredError:
			s.loginRequired = true
			return
		case *broadcastStoppedError:
			break
		}
	}

	s.endBroadcastAndPost()
}

func (s *Stream) endBroadcastAndPost() {
	if err := s.endBroadcast(); err != nil {
		log.Errorf("stream: %s: unable to end broadcast: %v", s.name, err)
		return
	}

	if s.config.IGTV.Enabled {
		if err := s.postToIGTV(); err != nil {
			log.Errorf("stream: %s: unable to post to IGTV: %v", s.name, err)
		}
	}

	if err := s.postToStories(); err != nil {
		log.Errorf("stream: %s: unable to post to Stories: %v", s.name, err)
		return
	}
}

func (s *Stream) cooldown() {
	log.Debugf("stream: %s: cooling down...", s.name)
	time.Sleep(cooldownDelay)
}

func (s *Stream) login() error {
	username := s.name
	token := s.config.Accounts[username].Token
	password := s.config.Accounts[username].Password

	i, err := s.loginByToken(username, token)
	if err == nil {
		s.instagram = i
		return nil
	}

	log.Error(err)
	log.Warnf("stream: %s: retrying login using password", username)

	i, err = s.loginByPassword(username, password)
	if err == nil {
		s.instagram = i
		if err := s.persistToken(); err != nil {
			return err
		}
		return nil
	}

	return err
}

func (s *Stream) loginByToken(username string, token string) (*instagram.Instagram, error) {
	log.Debugf("stream: %s: logging in by token", s.name)
	i, err := instagram.ImportFromString(token)
	if err != nil {
		return nil, fmt.Errorf("stream: %s: unable to login by token: %v", username, err)
	}

	log.Debugf("stream: %s: successfully logged in by token", s.name)
	return i, nil
}

func (s *Stream) loginByPassword(username string, password string) (*instagram.Instagram, error) {
	log.Debugf("stream: %s: logging in by password", s.name)
	i := instagram.New(username, password)
	if err := i.Login(); err != nil {
		return nil, err
	}

	log.Debugf("stream: %s: successfully logged in by password", s.name)
	return i, nil
}

func (s *Stream) respondChallenge() error {
	log.Debugf("stream: %s: processing challenge", s.name)
	err := s.instagram.Challenge.Process(s.apiPath)
	if err != nil {
		return fmt.Errorf("stream: %s: unable to process challenge: %v", s.name, err)
	}

	log.Debugf("stream: %s: waiting for security code", s.name)
	select {
	case <-time.After(challengeTimeout):
		return fmt.Errorf("stream: %s: timed out while waiting for challenge security code", s.name)
	case code := <-s.securityCode:
		log.Debugf("stream: %s: sending security code", s.name)
		err = s.instagram.Challenge.SendSecurityCode(code)
		if err != nil {
			return fmt.Errorf("stream: %s: unable to send security code: %v", s.name, err)
		}
	}

	log.Debugf("stream: %s: successfully responded to challenge", s.name)
	s.instagram.Account = s.instagram.Challenge.LoggedInUser
	if err := s.persistToken(); err != nil {
		return err
	}

	return nil
}

func (s *Stream) persistToken() error {
	log.Debugf("stream: %s: persisting token", s.name)
	newToken, err := instagram.ExportToString(s.instagram)
	if err != nil {
		return fmt.Errorf("stream: %s: unable to export new token: %v", s.name, err)
	}
	s.config.Accounts[s.name].Token = newToken

	if err := s.config.SaveConfig(); err != nil {
		return fmt.Errorf("stream: %s: unable to update config: %v", s.name, err)
	}

	log.Debugf("stream: %s: successfully persisted token in config file", s.name)
	return nil
}

func (s *Stream) createBroadcast(notify bool) error {
	log.Debugf("stream: %s: creating broadcast", s.name)
	live, err := s.instagram.Live.Create(streamWidth, streamHeight, s.config.Title)
	if err != nil {
		return err
	}
	if live.Status != "ok" {
		return fmt.Errorf("stream: %s: unable to create broadcast: %s", s.name, live.Status)
	}

	log.Debugf("stream: %s: starting broadcast %d", s.name, live.BroadcastID)
	start, err := s.instagram.Live.Start(live.BroadcastID, notify)
	if err != nil {
		return err
	}
	if start.Status != "ok" {
		return fmt.Errorf("stream: %s: unable to start broadcast %d: %s",
			s.name, live.BroadcastID, start.Status)
	}

	log.Debugf("stream: %s: unmuting comments in broadcast %d", s.name, live.BroadcastID)
	unmute, err := s.instagram.Live.UnmuteComment(live.BroadcastID)
	if err != nil {
		return err
	}
	if unmute.Status != "ok" {
		return fmt.Errorf("stream: %s: unable to unmute comments in broadcast %d: %s",
			s.name, live.BroadcastID, unmute.Status)
	}

	s.broadcastID = live.BroadcastID
	s.uploadURL = live.UploadURL
	s.startTime = time.Now()

	log.Infof("stream: %s: successfully started broadcast %d", s.name, s.broadcastID)
	return nil
}

func (s *Stream) runEncoder(ctx context.Context) error {
	var args []string
	args = append(args, "-i", s.config.InputURL)
	args = append(args, s.config.Encoder.Args...)
	args = append(args, "-f", "flv")
	args = append(args, s.uploadURL)

	cmd := exec.CommandContext(ctx, s.config.Encoder.Command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Debugf("stream: %s: starting encoder process", s.name)
	if err := cmd.Start(); err != nil {
		return err
	}
	log.Infof("stream: %s: encoder process started", s.name)

	if err := cmd.Wait(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			log.Debugf("stream: %s: encoder process killed by context cancellation", s.name)
			return nil
		}
		log.Errorf("stream: %s: encoder process error: %v", s.name, err)
		return err
	}
	return nil
}

func (s *Stream) heartbeatAndStatus() (*instagram.LiveHeartbeatAndGetViewerCountResponse, error) {
	log.Debugf("stream: %s: sending heartbeat and getting viewer count for broadcast %d", s.name, s.broadcastID)
	heartbeat, err := s.instagram.Live.HeartbeatAndGetViewerCount(s.broadcastID)
	if err != nil {
		return nil, err
	}
	if heartbeat.BroadcastStatus == "stopped" {
		se := &broadcastStoppedError{
			broadcastID: s.broadcastID,
		}
		log.Debugf("stream: %s: broadcast %d status is stopped", s.name, s.broadcastID)
		return heartbeat, se
	}

	log.Debugf("stream: %s: successfully sent heartbeat and got viewer count for broadcast %d",
		s.name, s.broadcastID)
	return heartbeat, nil
}

func (s *Stream) getComments(lastCommentTS int) (int, error) {
	log.Debugf("stream: %s: getting comments from broadcast %d", s.name, s.broadcastID)
	comments, err := s.instagram.Live.GetComment(s.broadcastID, numCommentsRequested, lastCommentTS)
	if err != nil {
		return lastCommentTS, err
	}
	newLastCommentTS := lastCommentTS

	for i := len(comments.Comments) - 1; i >= 0; i-- {
		comment := comments.Comments[i]
		log.Debugf("stream: %s: comment %d at %d from %s: %s",
			s.name, comment.PK, comment.CreatedAt, comment.User.Username, comment.Text)

		if comment.CreatedAt > newLastCommentTS {
			newLastCommentTS = comment.CreatedAt
		}
	}
	return newLastCommentTS, nil
}

func (s *Stream) endBroadcast() error {
	log.Debugf("stream: %s: ending broadcast %d", s.name, s.broadcastID)
	resp, err := s.instagram.Live.End(s.broadcastID, false)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		return fmt.Errorf("stream: %s: unable to end broadcast %d: %s", s.name, s.broadcastID, resp.Status)
	}

	log.Infof("stream: %s: successfully ended broadcast %d", s.name, s.broadcastID)
	return nil
}

func (s *Stream) postToStories() error {
	log.Debugf("stream: %s: posting broadcast %d to Stories", s.name, s.broadcastID)
	resp, err := s.instagram.Live.AddToPostLive(s.broadcastID)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		return fmt.Errorf("stream: %s: unable to post broadcast %d to Stories: %s",
			s.name, s.broadcastID, resp.Status)
	}

	log.Infof("stream: %s: successfully posted broadcast %d to Stories", s.name, s.broadcastID)
	return nil
}

func (s *Stream) postToIGTV() error {
	duration := time.Now().Sub(s.startTime)
	minDuration := time.Duration(s.config.IGTV.MinDuration) * time.Minute
	if duration < minDuration {
		return fmt.Errorf("stream: %s: broadcast duration is too short, will not post to IGTV", s.name)
	}

	log.Debugf("stream: %s: fetching thumbnail photos from broadcast %d", s.name, s.broadcastID)
	t, err := s.instagram.Live.GetPostLiveThumbnails(s.broadcastID)
	if err != nil {
		return err
	}

	// Grab a thumbnail near the middle of the video.
	idx := len(t.Thumbnails) / 2
	url := t.Thumbnails[idx]
	log.Debugf("stream: %s: downloading and converting thumbnail photo to JPEG", s.name)
	jpeg, err := s.instagram.GetThumbnailAsJPEG(url, jpegQuality)
	if err != nil {
		return err
	}

	log.Debugf("stream: %s: uploading thumbnail photo for IGTV", s.name)
	uploadID, err := s.instagram.UploadPhoto(bytes.NewReader(jpeg))
	if err != nil {
		return err
	}

	log.Debugf("stream: %s: posting broadcast %d to IGTV", s.name, s.broadcastID)
	igtv, err := s.instagram.Live.AddPostLiveToIGTV(
		s.broadcastID,
		uploadID,
		s.config.Title,
		s.config.IGTV.Description,
		s.config.IGTV.ShareToFeed,
	)
	if err != nil {
		return err
	}
	if igtv.Status != "ok" {
		return fmt.Errorf("stream: %s: unable to post broadcast %d to IGTV: %s",
			s.name, s.broadcastID, igtv.Status)
	}

	log.Infof("stream: %s: successfully posted broadcast %d to IGTV with ID: %d",
		s.name, s.broadcastID, igtv.IGTVPostID)
	return nil
}
