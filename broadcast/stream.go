package broadcast

import (
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
	procRestartDelay = 5 * time.Second
)

type FSMState uint32

const (
	idle FSMState = iota
	login
	challenge
	create
	upload
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
}

func NewStream(name string, config *Config) *Stream {
	s := &Stream{
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
	}

	return s
}

func (s *Stream) doIdle() {
	println("-------- in doIdle")

	log.Infof("stream: %s: idling...", s.name)
	time.Sleep(5 * time.Second)

	s.state = login
}

func (s *Stream) doLogin() {
	println("-------- in doLogin")

	err := s.login()
	if err == nil {
		log.Infof("stream: %s: logged in", s.instagram.Account.Username)
		s.state = create
		return
	}

	ce, ok := err.(instagram.ChallengeError)
	if !ok {
		log.Errorf("stream: %s: unable to login: %v", s.name, err)
		s.state = idle
		return
	}

	log.Warnf("stream: %s: challenge code required", s.name)
	s.apiPath = ce.Challenge.APIPath
	s.state = challenge
}

func (s *Stream) doChallenge() {
	println("-------- in doChallenge")

	err := s.respondChallenge()
	if err != nil {
		log.Errorf("stream: %s: unable to complete challenge: %v", s.name, err)
		s.state = idle
		return
	}

	s.state = create
}

func (s *Stream) doCreate() {
	println("-------- in doCreate")

	err := s.create(true)
	if err != nil {
		log.Errorf("stream: %s: unable to create live: %v", s.name, err)
		s.state = idle
		return
	}

	s.state = upload
}

func (s *Stream) doUpload() {
	println("-------- in doUpload")

	err := s.upload()
	if err != nil {
		log.Errorf("stream: %s: unable to upload: %v", s.name, err)
	}
}

func (s *Stream) doEnd() error {
	println("-------- in doEnd")

	err := s.end()
	if err != nil {
		return fmt.Errorf("stream: %s: unable to end stream: %v", s.name, err)
	}
	return nil
}

func (s *Stream) doShutdown() {
	println("-------- in doShutdown")

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
	}
}

func (s *Stream) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel

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

	err = s.instagram.Challenge.SendSecurityCode(<-s.securityCode)
	if err != nil {
		return fmt.Errorf("stream: %s: unable to send security code: %v", s.name, err)
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
	live, err := s.instagram.Live.Create(streamWidth, streamHeight, s.config.Message)
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

func (s *Stream) end() error {
	_, err := s.instagram.Live.End(s.broadcastID, true)
	if err != nil {
		return err
	}

	_, err = s.instagram.Live.AddToPostLive(s.broadcastID)
	if err != nil {
		return err
	}

	return nil
}
