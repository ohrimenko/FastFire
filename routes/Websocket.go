package routes

import (
	"bytes"
	"log"
	"net/http"
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

func (route Route) Websocket(router *mux.Router) {
	wsControllerMain := ws.NewControllerMain()
	ws := controllers.NewWebsocket(1000000, wsControllerMain.OnConnect, wsControllerMain.OnMessage, wsControllerMain.OnClose)

	router.Name("websocket.index").Methods("GET").Path("/chat").HandlerFunc(wsControllerMain.Index)

	router.Name("websocket.ws").Methods("GET").Path("/ws").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		wsClient, err := ws.NewWebsocketClient(conn)

		ws.Register(wsClient)

		defer func() {
			ws.Unregister(wsClient)
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
			ws.Broadcast(wsClient, message)
		}
	})
}
