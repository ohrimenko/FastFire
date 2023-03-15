package sse

import (
	"fmt"
	"net/http"

	"backnet/config"
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
		"views/layouts/main.html",
		"views/sse/index.html",
	}, 200, map[string]any{
		"Title":    "Data Sse",
		"SseHost":  config.Env("HOST"),
		"SsePort":  config.Env("SSE_PORT"),
		"SsesPort": config.Env("SSES_PORT"),
	})
}

func (сontroller ControllerMain) OnConnect(sseConn *controllers.SseConnection) {
	controllers.SseSendAll(fmt.Sprint("connection registered: ", sseConn.Key()))
}

func (сontroller ControllerMain) OnMessage(key string, data string) {
	controllers.SseSend(key, "send...")
	controllers.SseSendAll(data)
}

func (сontroller ControllerMain) OnClose(sseConn *controllers.SseConnection) {
	controllers.SseSendAll(fmt.Sprint("connection unregistered: ", sseConn.Key()))
}
