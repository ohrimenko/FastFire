package ws

import (
	"backnet/config"
	"fmt"
	"net/http"

	"backnet/controllers"
)

type ControllerMain struct {
	controllers.Controller
}

func NewControllerMain() ControllerMain {
	controller := ControllerMain{}

	return controller
}

func (сontroller ControllerMain) Index(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	request.View([]string{
		"views/websocket/layouts/main.html",
		"views/websocket/main/index.html",
	}, 200, map[string]any{
		"Title":   "websocket chat",
		"WsHost":  config.Env("HOST"),
		"WsPort":  config.Env("WS_PORT"),
		"WssPort": config.Env("WSS_PORT"),
	})
}

func (сontroller ControllerMain) OnConnect(wsClient *controllers.WebsocketClient) {
	controllers.WebsocketSendAll(fmt.Sprint("connection registered: ", wsClient.Key()))
}

func (сontroller ControllerMain) OnMessage(wsClient *controllers.WebsocketClient, message []byte) {
	controllers.WebsocketSend(wsClient.Key(), "send...")
	controllers.WebsocketSendAll(message)
}

func (сontroller ControllerMain) OnClose(wsClient *controllers.WebsocketClient) {
	controllers.WebsocketSendAll(fmt.Sprint("connection unregistered: ", wsClient.Key()))
}
