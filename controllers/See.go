package controllers

import (
	"backnet/components"
	"backnet/config"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bitly/go-simplejson"

	netsse "github.com/subchord/go-sse"
)

type ApiSse struct {
	Broker    *netsse.Broker
	Key       uint64
	EventI    uint64
	ConnectI  uint64
	OnConnect func(*SseConnection)
	OnMessage func(string, string)
	OnClose   func(*SseConnection)
	Valid     bool
}

type SseConnection struct {
	Connection *netsse.ClientConnection
}

var seeApp ApiSse

func NewSseConnection(conn *netsse.ClientConnection) *SseConnection {
	return &SseConnection{
		Connection: conn,
	}
}

func (api *ApiSse) UniqueEventId() string {
	atomic.AddUint64(&api.EventI, 1)

	return fmt.Sprintf("sse:%v:%v", api.Key, api.EventI)
}

func (api *ApiSse) UniqueConnectId() string {
	atomic.AddUint64(&api.ConnectI, 1)

	return fmt.Sprintf("sse:%v:%v:%v", api.Key, api.ConnectI, components.RandString(10))
}

func (api *ApiSse) SseHandler(writer http.ResponseWriter, request *http.Request) {
	// set the heartbeat interval to 1 minute
	client, err := api.Broker.ConnectWithHeartBeatInterval(api.UniqueConnectId(), writer, request, 1*time.Minute)
	if err != nil {
		log.Println(err)
		return
	}

	data := ""

	json := simplejson.New()
	json.Set("client_id", client.Id())

	payload, err := json.MarshalJSON()
	if err == nil {
		components.СonvertAssign(&data, payload)
	}

	api.Broker.Send(client.Id(), netsse.StringEvent{
		Id:    api.UniqueEventId(),
		Event: "connect",
		Data:  data,
	})

	if api.OnConnect != nil {
		api.OnConnect(NewSseConnection(client))
	}

	<-client.Done()

	if api.OnClose != nil {
		api.OnClose(NewSseConnection(client))
	}
}

func (api *ApiSse) api() *ApiSse {
	if !api.Valid {
		api.Broker = netsse.NewBroker(map[string]string{
			"Access-Control-Allow-Origin": "*",
		})

		api.Valid = true
		api.Key = uint64(time.Now().Unix())
	}

	return api
}

func SseApi() (*ApiSse, error) {
	if config.Env("SSE_SERVER_START") == "true" || config.Env("SSE_SERVER_START") == "1" || config.Env("SSES_SERVER_START") == "true" || config.Env("SSES_SERVER_START") == "1" {
		return seeApp.api(), nil
	}

	return nil, fmt.Errorf("See connection is prohibited on this server")
}

func SseUniqueEventId() string {
	sseApi, err := SseApi()

	if err == nil {
		return sseApi.UniqueEventId()
	}

	return ""
}

func SseOnMessage(w http.ResponseWriter, r *http.Request) {
	sseApi, err := SseApi()

	r.ParseForm()

	if err == nil {
		if r.Form.Get("client_id") != "" && r.Form.Get("data") != "" {
			splitKey := strings.Split(r.Form.Get("client_id"), ":")

			if len(splitKey) == 4 {
				if splitKey[0] == "sse" {
					if sseKey, err := strconv.ParseUint(splitKey[1], 10, 64); err == nil {
						if sseKey == sseApi.Key {
							if sseApi.Broker.IsClientPresent(splitKey[0] + ":" + splitKey[1] + ":" + splitKey[2] + ":" + splitKey[3]) {
								if sseApi.OnMessage != nil {
									sseApi.OnMessage(splitKey[0]+":"+splitKey[1]+":"+splitKey[2]+":"+splitKey[3], r.Form.Get("data"))
								}
							}
						}
					}
				}
			}
		}
	}
}

func (api *ApiSse) Send(key string, data string) {
	splitKey := strings.Split(key, ":")

	if len(splitKey) == 4 {
		if splitKey[0] == "sse" {
			if sseKey, err := strconv.ParseUint(splitKey[1], 10, 64); err == nil {
				if sseKey == api.Key {
					api.Broker.Send(splitKey[0]+":"+splitKey[1]+":"+splitKey[2]+":"+splitKey[3], netsse.StringEvent{
						Id:    SseUniqueEventId(),
						Event: "message",
						Data:  data,
					})
				}
			}
		}
	}
}

func (api *ApiSse) SendAll(data string) {
	api.Broker.Broadcast(netsse.StringEvent{
		Id:    SseUniqueEventId(),
		Event: "message",
		Data:  data,
	})
}

func (sConn *SseConnection) Key() string {
	return sConn.Connection.Id()
}

func (sConn *SseConnection) Send(key string, data string) {
	sseApi, err := SseApi()

	if err == nil {
		sseApi.Send(key, data)
	}
}

func (sConn *SseConnection) SendAll(data string) {
	sseApi, err := SseApi()

	if err == nil {
		sseApi.SendAll(data)
	}
}

func SseSend(key string, data string) {
	sseApi, err := SseApi()

	if err == nil {
		sseApi.Send(key, data)
	}
}

func SseSendAll(data string) {
	sseApi, err := SseApi()

	if err == nil {
		sseApi.SendAll(data)
	}
}
