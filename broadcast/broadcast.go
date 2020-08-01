package broadcast

import (
	"context"
	"github.com/sbekti/broadcastd/config"
	"github.com/sbekti/broadcastd/instagram"
	"github.com/sbekti/broadcastd/process"
	"golang.org/x/sync/errgroup"
	"strconv"
)

type Broadcast struct {
	Live    bool
	Outputs map[string]*Output
	config  *config.Config
	cancel  context.CancelFunc
	done    chan error
}

type Output struct {
	Instagram *instagram.Instagram
	Process   *process.Process
}

func NewBroadcast(c *config.Config) *Broadcast {
	b := &Broadcast{
		Live:    false,
		Outputs: map[string]*Output{},
		config:  c,
		cancel:  nil,
	}

	for username := range c.Accounts {
		var port int

		if username == "shbekti_test" {
			port = 1936
		} else {
			port = 1937
		}
		outputURL := "rtmp://localhost:" + strconv.Itoa(port) + "/live/" + username

		var args []string
		args = append(args, "-i", c.InputURL)
		args = append(args, c.Encoder.Args...)
		args = append(args, "-f", "flv", outputURL)

		b.Outputs[username] = &Output{
			Instagram: nil,
			Process: &process.Process{
				Command:     c.Encoder.Path,
				Args:        args,
				AutoRestart: true,
			},
		}
	}

	return b
}

func (b *Broadcast) startProcesses(ctx context.Context) error {
	g, errCtx := errgroup.WithContext(ctx)

	for _, output := range b.Outputs {
		output := output
		g.Go(func() error {
			return output.Process.Run(errCtx)
		})
	}

	return g.Wait()
}

func (b *Broadcast) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	done := make(chan error, 1)

	go func() {
		done <- b.startProcesses(ctx)
	}()

	b.Live = true
	b.done = done
}

func (b *Broadcast) Stop() error {
	b.cancel()
	err := <-b.done
	b.Live = false
	return err
}
