package server

import (
	"github.com/labstack/echo/v4"
	"github.com/sbekti/broadcastd/broadcast"
)

type BroadcastContext struct {
	echo.Context
	*broadcast.Broadcast
}

func stateMiddleware(b *broadcast.Broadcast) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			bc := &BroadcastContext{c, b}
			return next(bc)
		}
	}
}
