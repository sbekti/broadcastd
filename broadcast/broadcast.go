package broadcast

import (
	"context"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/sbekti/broadcastd/config"
	"github.com/sbekti/broadcastd/instagram"
	"github.com/sbekti/broadcastd/stream"
	"golang.org/x/sync/errgroup"
)

const (
	width  = 720
	height = 1280
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
	Stream    *stream.Stream
}

func NewBroadcast(c *config.Config) *Broadcast {
	b := &Broadcast{
		Live:    false,
		Outputs: map[string]*Output{},
		config:  c,
		cancel:  nil,
		done:    nil,
	}

	for username := range c.Accounts {
		var args []string
		args = append(args, "-i", c.InputURL)
		args = append(args, c.Encoder.Args...)
		args = append(args, "-f", "flv")

		b.Outputs[username] = &Output{
			Instagram: nil,
			Stream: &stream.Stream{
				Status:      "",
				Command:     c.Encoder.Path,
				Args:        args,
				UploadURL:   "",
				BroadcastID: -1,
				AutoRestart: true,
			},
		}
	}

	return b
}

func (b *Broadcast) Login() error {
	g, _ := errgroup.WithContext(context.Background())

	for username, output := range b.Outputs {
		username := username
		output := output

		g.Go(func() error {
			token := b.config.Accounts[username].Token
			i, err := instagram.ImportFromString(token)
			if err != nil {
				return fmt.Errorf("unable to login as %s: %v", username, err)
			}

			log.Infof("Logged in as %s", i.Account.Username)
			output.Instagram = i
			return nil
		})
	}

	return g.Wait()
}

func (b *Broadcast) startStreams(ctx context.Context) error {
	g, errCtx := errgroup.WithContext(ctx)

	for _, output := range b.Outputs {
		output := output

		g.Go(func() error {
			live, err := output.Instagram.Live.Create(width, height, b.config.Message)
			if err != nil {
				return err
			}

			output.Stream.BroadcastID = live.BroadcastID
			output.Stream.UploadURL = live.UploadURL

			_, err = output.Instagram.Live.Start(live.BroadcastID, false)
			if err != nil {
				return err
			}

			_, err = output.Instagram.Live.UnmuteComment(live.BroadcastID)
			if err != nil {
				return err
			}

			return output.Stream.Run(errCtx)
		})
	}

	return g.Wait()
}

func (b *Broadcast) stopStreams() error {
	var g errgroup.Group

	for _, output := range b.Outputs {
		output := output

		g.Go(func() error {
			broadcastID := output.Stream.BroadcastID

			_, err := output.Instagram.Live.End(broadcastID, true)
			if err != nil {
				return err
			}

			_, err = output.Instagram.Live.AddToPostLive(broadcastID)
			if err != nil {
				return err
			}

			return nil
		})
	}

	return g.Wait()
}

func (b *Broadcast) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	done := make(chan error, 1)

	go func() {
		done <- b.startStreams(ctx)
	}()

	b.Live = true
	b.done = done
}

func (b *Broadcast) Stop() error {
	var g errgroup.Group

	g.Go(func() error {
		return b.stopStreams()
	})

	g.Go(func() error {
		b.cancel()
		return <-b.done
	})

	err := g.Wait()
	b.Live = false
	return err
}

func (b *Broadcast) Test() {
	for _, output := range b.Outputs {
		if err := output.Stream.Kill(); err != nil {
			log.Error(err)
		}
	}
}
