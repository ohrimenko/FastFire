package ws

import (
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
		"Title": "websocket chat",
	})
}

func (сontroller ControllerMain) OnConnect(wsClient *controllers.WebsocketClient) {
	wsClient.SendAll(fmt.Sprint("connection registered: ", wsClient.Key()))
	wsClient.Key()
}

func (сontroller ControllerMain) OnMessage(wsClient *controllers.WebsocketClient, message []byte) {
	wsClient.Send(wsClient.Key(), "send...")
	wsClient.SendAll(message)
	wsClient.Key()
}

func (сontroller ControllerMain) OnClose(wsClient *controllers.WebsocketClient) {
	wsClient.SendAll(fmt.Sprint("connection unregistered: ", wsClient.Key()))
	wsClient.Key()
}
