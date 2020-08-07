package broadcast

import (
	"github.com/labstack/echo/v4"
)

type StateContext struct {
	echo.Context
	*Broadcast
}

func stateMiddleware(b *Broadcast) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			bc := &StateContext{c, b}
			return next(bc)
		}
	}
}
