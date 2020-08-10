package broadcast

import (
	"fmt"
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

type postSecurityCodeReq struct {
	Account      string `json:"account"`
	SecurityCode string `json:"security_code"`
}

type postSecurityCodeRes struct {
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

func PostSecurityCode(c echo.Context) error {
	req := new(postSecurityCodeReq)
	if err := c.Bind(req); err != nil {
		return err
	}

	sc := c.(*StateContext)

	if _, ok := sc.streams[req.Account]; !ok {
		return c.JSON(http.StatusBadRequest, postSecurityCodeRes{
			Status: "error",
			Error:  fmt.Sprintf("account %s does not exist", req.Account),
		})
	}

	err := sc.streams[req.Account].PutSecurityCode(req.SecurityCode)
	if err != nil {
		return c.JSON(http.StatusBadRequest, postSecurityCodeRes{
			Status: "error",
			Error:  err.Error(),
		})
	} else {
		return c.JSON(http.StatusOK, postSecurityCodeRes{
			Status: "ok",
			Error:  "",
		})
	}
}
