package process

import (
	"context"
	"os"
	"os/exec"
	"time"
)

const (
	gracePeriod  = 5 * time.Second
	restartDelay = 5 * time.Second
)

type Process struct {
	Status      string
	Command     string
	Args        []string
	AutoRestart bool
}

func (p *Process) Run(ctx context.Context) error {
	name, err := exec.LookPath(p.Command)
	if err != nil {
		return err
	}

	cmd := exec.Command(name, p.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan error, 1)

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-time.After(gracePeriod):
			return cmd.Process.Kill()
		case <-done:
			return nil
		}
	case err = <-done:
		if p.AutoRestart {
			time.Sleep(restartDelay)
			return p.Run(ctx)
		}
	}

	if err != nil {
		return err
	}
	return nil
}
