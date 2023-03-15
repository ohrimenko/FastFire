package routes

import (
	"github.com/gorilla/mux"

	"backnet/controllers"
	"backnet/controllers/sse"
)

func (route Route) Sse(router *mux.Router) {
	sseControllerMain := sse.NewControllerMain()

	sseApi, err := controllers.SseApi()

	sseApi.OnConnect = sseControllerMain.OnConnect
	sseApi.OnMessage = sseControllerMain.OnMessage
	sseApi.OnClose = sseControllerMain.OnClose

	if err == nil {
		router.Name("sse.connect").Methods("GET").Path("/sse").HandlerFunc(sseApi.SseHandler)
	}
}
