package controllers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"backnet/config"

	"github.com/gorilla/websocket"
)

type WebsocketClient struct {
	// The websocket connection.
	Connect *websocket.Conn

	Time int64

	WsKey     uint64
	StackKey  uint64
	ClientKey uint64
}

func NewWebsocketClient(connect *websocket.Conn, wsKey uint64, stackKey uint64, clientKey uint64) (*WebsocketClient, error) {
	var err error

	if connect == nil {
		err = errors.New("No WebsocketStack")
	}

	wsClient := &WebsocketClient{
		Connect:   connect,
		WsKey:     wsKey,
		StackKey:  stackKey,
		ClientKey: clientKey,
	}

	return wsClient, err
}

func (wsClient *WebsocketClient) Key() string {
	return fmt.Sprintf("%d:%d:%d", wsClient.WsKey, wsClient.StackKey, wsClient.ClientKey)
}

func (wsClient *WebsocketClient) SendAll(message any) {
	if ws, ok := Websockets[wsClient.WsKey]; ok {
		ws.SendAll(message)
	}
}

func (wsClient *WebsocketClient) Send(key string, message any) {
	if ws, ok := Websockets[wsClient.WsKey]; ok {
		splitKey := strings.Split(key, ":")

		if len(splitKey) == 3 {
			var wsKey uint64
			var stackKey uint64
			var clientKey uint64

			if n, err := strconv.ParseUint(splitKey[0], 10, 64); err == nil {
				wsKey = n
				if n, err := strconv.ParseUint(splitKey[1], 10, 64); err == nil {
					stackKey = n
					if n, err := strconv.ParseUint(splitKey[2], 10, 64); err == nil {
						clientKey = n

						ws.Send(wsKey, stackKey, clientKey, message)
					}
				}
			}
		}
	}
}

func (wsClient *WebsocketClient) Write(message []byte) bool {
	wsClient.Connect.SetWriteDeadline(time.Now().Add(config.WriteWait))

	writer, err := wsClient.Connect.NextWriter(websocket.TextMessage)
	if err == nil {
		_, err = writer.Write(message)

		err = writer.Close()
	}

	if err == nil {
		atomic.SwapInt64(&wsClient.Time, time.Now().Unix())

		return true
	} else {
		return false
	}
}
