package broadcast

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

const (
	shutdownGracePeriod = 5 * time.Second
)

type Server struct {
	IP   string
	Port int
	b    *Broadcast
	e    *echo.Echo
}

func NewServer(b *Broadcast, ip string, port int) *Server {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	e.HideBanner = true

	e.Logger = logrusLogger{Logger: logrus.StandardLogger()}
	e.Use(loggerHook())
	e.Use(stateMiddleware(b))
	e.Use(middleware.Recover())

	e.GET("/", GetIndex)

	g := e.Group("/api/v1")
	g.POST("/live", PostLive)
	g.POST("/security_code", PostSecurityCode)

	return &Server{
		IP:   ip,
		Port: port,
		e:    e,
	}
}

func (s *Server) Start() error {
	port := strconv.Itoa(s.Port)
	server := &http.Server{
		Addr:         s.IP + ":" + port,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	return s.e.StartServer(server)
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()
	return s.e.Shutdown(ctx)
}
