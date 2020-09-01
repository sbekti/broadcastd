package broadcast

import (
	"fmt"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
	"net/http"
	"sort"
)

type postLiveReq struct {
	Live bool `json:"live"`
}

type postLiveRes struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

type indexRes struct {
	Input   statusInfo
	Outputs []outputInfo
}

type statusInfo struct {
	Live bool
}

type outputInfo struct {
	Name              string
	Status            string
	ChallengeRequired bool
}

type getSecurityCodeRes struct {
	Account string
}

type postSecurityCodeRes struct {
	Account string
	Status  string
	Error   string
}

func GetIndex(c echo.Context) error {
	sc := c.(*StateContext)

	var keys []string
	for k := range sc.streams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var outputs []outputInfo
	for _, key := range keys {
		outputs = append(outputs, outputInfo{
			Name:              sc.streams[key].name,
			Status:            sc.streams[key].status,
			ChallengeRequired: sc.streams[key].status == challengeRequired,
		})
	}

	data := &indexRes{
		Input: statusInfo{
			Live: sc.streaming,
		},
		Outputs: outputs,
	}

	return c.Render(http.StatusOK, "index", data)
}

func GetSecurityCode(c echo.Context) error {
	account := c.Param("account")

	data := &getSecurityCodeRes{
		Account: account,
	}

	return c.Render(http.StatusOK, "security_code", data)
}

func PostSecurityCode(c echo.Context) error {
	account := c.FormValue("account")
	securityCode := c.FormValue("security_code")

	sc := c.(*StateContext)

	if _, ok := sc.streams[account]; !ok {
		return c.JSON(http.StatusBadRequest, postSecurityCodeRes{
			Status: "error",
			Error:  fmt.Sprintf("account %s does not exist", account),
		})
	}

	err := sc.streams[account].PutSecurityCode(securityCode)
	if err != nil {
		return c.JSON(http.StatusBadRequest, postSecurityCodeRes{
			Status: "error",
			Error:  err.Error(),
		})
	} else {
		return c.Redirect(http.StatusSeeOther, "/")
	}
}

func GetComments(c echo.Context) error {
	return c.Render(http.StatusOK, "comments", nil)
}

func WebSocketComments(c echo.Context) error {
	sc := c.(*StateContext)

	websocket.Handler(func(ws *websocket.Conn) {
		defer func(conn *websocket.Conn) {
			sc.connectionsMux.Lock()
			delete(sc.connections, conn)
			conn.Close()
			sc.connectionsMux.Unlock()
		}(ws)

		sc.connectionsMux.Lock()
		sc.connections[ws] = struct{}{}
		sc.connectionsMux.Unlock()

		msg := ""
		for {
			if err := websocket.Message.Receive(ws, &msg); err != nil {
				return
			}
			log.Debugf("ws: received: %s", msg)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
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
