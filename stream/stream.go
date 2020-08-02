package stream

import (
	"context"
	"github.com/labstack/gommon/log"
	"os"
	"os/exec"
	"time"
)

const (
	gracePeriod  = 5 * time.Second
	restartDelay = 5 * time.Second
)

type Stream struct {
	Status      string
	Command     string
	Args        []string
	UploadURL   string
	BroadcastID int
	AutoRestart bool
	process     *os.Process
}

func (s *Stream) Run(ctx context.Context) error {
	name, err := exec.LookPath(s.Command)
	if err != nil {
		return err
	}

	args := s.Args
	args = append(args, s.UploadURL)

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan error, 1)

	if err := cmd.Start(); err != nil {
		return err
	}
	s.process = cmd.Process

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-time.After(gracePeriod):
			log.Errorf("process %s still did not exit, sending a SIGKILL", name)
			return cmd.Process.Kill()
		case <-done:
			log.Infof("process %s gracefully exited", name)
			return nil
		}
	case err = <-done:
		if s.AutoRestart {
			log.Errorf("process %s exited and will be restarted", name)
			time.Sleep(restartDelay)
			return s.Run(ctx)
		}
	}

	if err != nil {
		return err
	}
	return nil
}

func (s *Stream) Kill() error {
	return s.process.Kill()
}
