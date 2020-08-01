package server

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type postLiveReq struct {
	Live bool `json:"live"`
}

type postLiveRes struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func PostLive(c echo.Context) error {
	req := new(postLiveReq)
	if err := c.Bind(req); err != nil {
		return err
	}

	bc := c.(*BroadcastContext)
	if bc.Live == req.Live {
		var errorMsg string
		if bc.Live {
			errorMsg = "Broadcast is already live."
		} else {
			errorMsg = "Broadcast is currently not live."
		}

		res := postLiveRes{
			Status: "error",
			Error:  errorMsg,
		}
		return c.JSON(http.StatusBadRequest, res)
	}

	if req.Live {
		bc.Broadcast.Start()
	} else {
		bc.Broadcast.Stop()
	}

	res := postLiveRes{
		Status: "ok",
		Error:  "",
	}
	return c.JSON(http.StatusOK, res)
}
