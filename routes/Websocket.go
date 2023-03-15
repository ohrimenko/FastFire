package routes

import (
	"bytes"
	"log"
	"net/http"
	"strings"
	"time"

	"backnet/config"
	"backnet/controllers"
	"backnet/controllers/ws"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var originHosts []string = []string{}

var wsCtrl *controllers.Websocket

func (route Route) Websocket(router *mux.Router) {
	if config.Env("WS_CHECK_ORIGN") == "true" || config.Env("WS_CHECK_ORIGN") == "1" {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	} else if len(originHosts) > 0 {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			for i, _ := range originHosts {
				if originHosts[i] == r.Header.Get("Origin") {
					return true
				}
			}

			return false
		}
	}

	wsControllerMain := ws.NewControllerMain()
	if wsCtrl == nil {
		wsCtrl = controllers.NewWebsocket(1000000, wsControllerMain.OnConnect, wsControllerMain.OnMessage, wsControllerMain.OnClose)

		if !(config.Env("WS_CHECK_ORIGN") == "true" || config.Env("WS_CHECK_ORIGN") == "false" || config.Env("WS_CHECK_ORIGN") == "1" || config.Env("WS_CHECK_ORIGN") == "0") && len(config.Env("WS_CHECK_ORIGN")) > 0 {
			originHosts = strings.Split(config.Env("WS_CHECK_ORIGN"), ",")
		}
	}

	router.Name("websocket.ws").Methods("GET").Path("/ws").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		wsClient, err := wsCtrl.NewWebsocketClient(conn)

		wsCtrl.Register(wsClient)

		defer func() {
			wsCtrl.Unregister(wsClient)
			wsClient.Connect.Close()
		}()

		wsClient.Connect.SetReadLimit(config.MaxMessageSize)
		wsClient.Connect.SetReadDeadline(time.Now().Add(config.PongWait))
		wsClient.Connect.SetPongHandler(func(string) error { wsClient.Connect.SetReadDeadline(time.Now().Add(config.PongWait)); return nil })
		for {
			_, message, err := wsClient.Connect.ReadMessage()
			if err != nil {
				//fmt.Println("ERROR ReadMessage: ", err)

				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("error: %v", err)
				}
				break
			}
			message = bytes.TrimSpace(bytes.Replace(message, []byte{'\n'}, []byte{' '}, -1))
			wsCtrl.Broadcast(wsClient, message)
		}
	})
}
