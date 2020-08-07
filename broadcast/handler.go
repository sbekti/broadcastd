package broadcast

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

type postLiveReq struct {
	Live bool `json:"live"`
}

type postLiveRes struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func GetIndex(c echo.Context) error {
	time.Sleep(5 * time.Second)
	return c.String(http.StatusOK, "Hello, world!")
}

func PostLive(c echo.Context) error {
	req := new(postLiveReq)
	if err := c.Bind(req); err != nil {
		return err
	}

	sc := c.(*StateContext)
	var err error

	if req.Live {
		err = sc.Broadcast.StartStreams()
	} else {
		err = sc.Broadcast.StopStreams()
	}

	if err != nil {
		return c.JSON(http.StatusBadRequest, postLiveRes{
			Status: "error",
			Error:  err.Error(),
		})
	} else {
		return c.JSON(http.StatusOK, postLiveRes{
			Status: "ok",
			Error:  "",
		})
	}
}
