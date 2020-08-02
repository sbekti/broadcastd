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

		return c.JSON(http.StatusBadRequest, postLiveRes{
			Status: "error",
			Error:  errorMsg,
		})
	}

	if req.Live {
		bc.Broadcast.Start()
		return c.JSON(http.StatusOK, postLiveRes{
			Status: "ok",
			Error:  "",
		})
	} else {
		err := bc.Broadcast.Stop()
		if err != nil {
			return c.JSON(http.StatusOK, postLiveRes{
				Status: "error",
				Error:  err.Error(),
			})
		}

		return c.JSON(http.StatusOK, postLiveRes{
			Status: "ok",
			Error:  "",
		})
	}

}
