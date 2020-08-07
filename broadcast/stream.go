package broadcast

import (
	"bytes"
	"context"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/sbekti/broadcastd/instagram"
	"os"
	"os/exec"
	"time"
)

const (
	streamWidth      = 720
	streamHeight     = 1280
	cooldownDelay    = 30 * time.Second
	procRestartDelay = 5 * time.Second
	heartbeatDelay   = 5 * time.Second
	challengeTimeout = 2 * time.Minute
	jpegQuality      = 95
)

type FSMState uint32

const (
	idle FSMState = iota
	login
	challenge
	create
	upload
	cooldown
)

type Stream struct {
	name         string
	config       *Config
	instagram    *instagram.Instagram
	broadcastID  int
	uploadURL    string
	apiPath      string
	securityCode chan string
	ctx          context.Context
	cancel       context.CancelFunc
	state        FSMState
	done         chan error
	startTime    time.Time
}

type streamStoppedError struct {
	broadcastID int
}

func (e streamStoppedError) Error() string {
	return fmt.Sprintf("stream %d has stopped", e.broadcastID)
}

func NewStream(name string, config *Config) *Stream {
	var s = &Stream{
		name:         name,
		config:       config,
		instagram:    nil,
		broadcastID:  0,
		uploadURL:    "",
		apiPath:      "",
		securityCode: make(chan string),
		ctx:          nil,
		cancel:       nil,
		state:        idle,
		done:         make(chan error, 1),
		startTime:    time.Time{},
	}

	return s
}

func (s *Stream) doIdle() {
	log.Infof("stream: %s: idling...", s.name)
	time.Sleep(5 * time.Second)
}

func (s *Stream) doLogin() {
	err := s.login()
	if err == nil {
		log.Infof("stream: %s: logged in", s.instagram.Account.Username)
		s.state = create
		return
	}

	ce, ok := err.(instagram.ChallengeError)
	if !ok {
		log.Errorf("stream: %s: unable to login: %v", s.name, err)
		s.state = cooldown
		return
	}

	log.Warnf("stream: %s: challenge code required", s.name)
	s.apiPath = ce.Challenge.APIPath
	s.state = challenge
}

func (s *Stream) doChallenge() {
	err := s.respondChallenge()
	if err != nil {
		log.Errorf("stream: %s: unable to complete challenge: %v", s.name, err)
		s.state = cooldown
		return
	}

	s.state = create
}

func (s *Stream) doCreate() {
	err := s.create(true)
	if err != nil {
		log.Errorf("stream: %s: unable to create live: %v", s.name, err)
		s.state = cooldown
		return
	}

	s.state = upload
}

func (s *Stream) doUpload() {
	err := s.upload()
	if err != nil {
		log.Errorf("stream: %s: unable to upload: %v", s.name, err)
	}
}

func (s *Stream) doCooldown() {
	log.Infof("stream: %s: cooling down...", s.name)
	time.Sleep(cooldownDelay)

	s.state = login
}

func (s *Stream) doEnd() error {
	err := s.end()
	if err != nil {
		return fmt.Errorf("stream: %s: unable to end stream: %v", s.name, err)
	}
	return nil
}

func (s *Stream) doShutdown() {
	if s.state == upload {
		s.state = idle
		s.done <- s.doEnd()
		return
	}

	s.state = idle
	s.done <- nil
}

func (s *Stream) stateMachine() {
	switch s.state {
	case idle:
		s.doIdle()
	case login:
		s.doLogin()
	case challenge:
		s.doChallenge()
	case create:
		s.doCreate()
	case upload:
		s.doUpload()
	case cooldown:
		s.doCooldown()
	}
}

func (s *Stream) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
	s.state = login

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				s.doShutdown()
				return
			default:
				s.stateMachine()
			}
		}
	}()

	return nil
}

func (s *Stream) Stop() error {
	s.cancel()
	return <-s.done
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
	i, err := instagram.ImportFromString(token)
	if err != nil {
		return nil, fmt.Errorf("stream: %s: unable to login using token: %v", username, err)
	}
	return i, nil
}

func (s *Stream) loginByPassword(username string, password string) (*instagram.Instagram, error) {
	i := instagram.New(username, password)
	if err := i.Login(); err != nil {
		return i, err
	}
	return i, nil
}

func (s *Stream) respondChallenge() error {
	err := s.instagram.Challenge.Process(s.apiPath)
	if err != nil {
		return fmt.Errorf("stream: %s: unable to process challenge: %v", s.name, err)
	}

	select {
	case <-time.After(challengeTimeout):
		return fmt.Errorf("stream: %s: timed out while waiting for challenge security code", s.name)
	case code := <-s.securityCode:
		err = s.instagram.Challenge.SendSecurityCode(code)
		if err != nil {
			return fmt.Errorf("stream: %s: unable to send security code: %v", s.name, err)
		}
	}

	s.instagram.Account = s.instagram.Challenge.LoggedInUser
	if err := s.persistToken(); err != nil {
		return err
	}
	return nil
}

func (s *Stream) persistToken() error {
	newToken, err := instagram.ExportToString(s.instagram)
	if err != nil {
		return fmt.Errorf("stream: %s: unable to export new token: %v", s.name, err)
	}

	s.config.Accounts[s.name].Token = newToken

	if err := s.config.SaveConfig(); err != nil {
		return fmt.Errorf("stream: %s: unable to update config: %v", s.name, err)
	}
	return nil
}

func (s *Stream) create(notify bool) error {
	live, err := s.instagram.Live.Create(streamWidth, streamHeight, s.config.Title)
	if err != nil {
		return err
	}

	_, err = s.instagram.Live.Start(live.BroadcastID, notify)
	if err != nil {
		return err
	}

	_, err = s.instagram.Live.UnmuteComment(live.BroadcastID)
	if err != nil {
		return err
	}

	s.broadcastID = live.BroadcastID
	s.uploadURL = live.UploadURL
	s.startTime = time.Now()
	return nil
}

func (s *Stream) upload() error {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	var args []string
	args = append(args, "-i", s.config.InputURL)
	args = append(args, s.config.Encoder.Args...)
	args = append(args, "-f", "flv")
	args = append(args, s.uploadURL)

	cmd := exec.Command(s.config.Encoder.Command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)

	go func() {
		done <- cmd.Wait()
	}()

	go func() {
		lastCommentTS := 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(heartbeatDelay):
				err, newLastCommentTS := s.monitor(lastCommentTS)
				if err == nil {
					lastCommentTS = newLastCommentTS
					continue
				}

				se, ok := err.(streamStoppedError)
				if !ok {
					log.Errorf("stream: %s: %v", s.name, err)
					continue
				}

				log.Warnf("stream: %s: broadcast %d has stopped", s.name, se.broadcastID)
				err = s.doEnd()
				if err != nil {
					log.Errorf("stream: %s: %v", s.name, err)
				}

				s.state = login
				cancel()
			}
		}
	}()

	select {
	case <-ctx.Done():
		log.Infof("stream: %s: killing process", s.name)
		return cmd.Process.Kill()
	case <-done:
		log.Errorf("stream: %s: process exited and will be restarted", s.name)
		time.Sleep(procRestartDelay)
		return fmt.Errorf("stream: %s: process exited and will be restarted", s.name)
	}
}

func (s *Stream) monitor(lastCommentTS int) (error, int) {
	heartbeat, err := s.instagram.Live.HeartbeatAndGetViewerCount(s.broadcastID)
	if err != nil {
		return err, lastCommentTS
	}
	log.Infof("stream: %s: broadcast status: %s", s.name, heartbeat.BroadcastStatus)

	if heartbeat.BroadcastStatus == "stopped" {
		se := streamStoppedError{
			broadcastID: s.broadcastID,
		}
		return se, lastCommentTS
	}

	comments, err := s.instagram.Live.GetComment(s.broadcastID, 10, lastCommentTS)
	if err != nil {
		return err, lastCommentTS
	}

	newLastCommentTS := lastCommentTS

	for i := len(comments.Comments) - 1; i >= 0; i-- {
		comment := comments.Comments[i]
		log.Warnf("stream: %s: comment %d at %d %s: %s",
			s.name, comment.PK, comment.CreatedAt, comment.User.Username, comment.Text)

		if comment.CreatedAt > newLastCommentTS {
			newLastCommentTS = comment.CreatedAt
			log.Warnf("updated newLastCommentTS to: %d", newLastCommentTS)
		}
	}
	return nil, newLastCommentTS
}

func (s *Stream) end() error {
	_, err := s.instagram.Live.End(s.broadcastID, true)
	if err != nil {
		return err
	}

	// Posting to Stories and IGTV are mutually exclusive.
	// If IGTV is not enabled then post live to Stories instead.
	if !s.config.IGTV.Enabled {
		_, err = s.instagram.Live.AddToPostLive(s.broadcastID)
		if err != nil {
			return err
		}
	}

	duration := time.Now().Sub(s.startTime)
	minDuration := time.Duration(s.config.IGTV.MinDuration) * time.Minute
	if duration < minDuration {
		log.Warnf("stream: live duration is too short, will not post to IGTV")
		return nil
	}

	t, err := s.instagram.Live.GetPostLiveThumbnails(s.broadcastID)
	if err != nil {
		return err
	}

	// Grab a thumbnail near the middle of the video.
	thumbIndex := len(t.Thumbnails) / 2
	thumbURL := t.Thumbnails[thumbIndex]
	jpegThumb, err := s.instagram.GetThumbnailAsJPEG(thumbURL, jpegQuality)
	if err != nil {
		return err
	}

	uploadID, err := s.instagram.UploadPhoto(bytes.NewReader(jpegThumb))
	if err != nil {
		return err
	}

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

	log.Infof("stream: post to IGTV: %+v", igtv)

	return nil
}
