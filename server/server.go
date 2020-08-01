package server

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/sbekti/broadcastd/broadcast"
	"golang.org/x/sync/errgroup"
	"strconv"
	"time"
)

const (
	gracePeriod = 5 * time.Second
)

type Server struct {
	IP        string
	Port      int
	Broadcast *broadcast.Broadcast
	e         *echo.Echo
}

func NewServer(ip string, port int, b *broadcast.Broadcast) *Server {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	e.HideBanner = true

	e.Use(stateMiddleware(b))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", GetIndex)

	g := e.Group("/api/v1")
	g.POST("/live", PostLive)

	return &Server{
		IP:        ip,
		Port:      port,
		Broadcast: b,
		e:         e,
	}
}

func (s *Server) Start() error {
	port := strconv.Itoa(s.Port)
	return s.e.Start(s.IP + ":" + port)
}

func (s *Server) Shutdown() error {
	var g errgroup.Group

	g.Go(func() error {
		if s.Broadcast.Live {
			return s.Broadcast.Stop()
		}
		return nil
	})

	g.Go(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), gracePeriod)
		defer cancel()
		return s.e.Shutdown(ctx)
	})

	return g.Wait()
}
