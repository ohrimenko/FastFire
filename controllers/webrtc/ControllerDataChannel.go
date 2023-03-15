package webrtc

import (
	"backnet/controllers"
	"fmt"
)

type ControllerDataChannel struct {
	controllers.Controller
}

func NewControllerDataChannel() ControllerDataChannel {
	controller := ControllerDataChannel{}

	return controller
}

func (controller ControllerDataChannel) OnConnect(wrConn *webrtConnection) {
	WebrtcSendAll(fmt.Sprint("connection registered: ", wrConn.Key()))
}

func (controller ControllerDataChannel) OnMessage(wrConn *webrtConnection, data []byte) {
	WebrtcSend(wrConn.Key(), "send...")
	WebrtcSendAll(data)
}

func (controller ControllerDataChannel) OnClose(wrConn *webrtConnection) {
	WebrtcSendAll(fmt.Sprint("connection unregistered: ", wrConn.Key()))
}
