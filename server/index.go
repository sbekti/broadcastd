package server

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func GetIndex(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, world!")
}
