package controllers

import (
	"backnet/components"
)

type BroadcastWebsocket struct {
	WsKey     uint64
	StackKey  uint64
	ClientKey uint64
	Message   []byte
}

func NewBroadcastWebsocket(wsKey uint64, stackKey uint64, clientKey uint64, message any) *BroadcastWebsocket {
	wsBroadcast := &BroadcastWebsocket{
		WsKey:     wsKey,
		StackKey:  stackKey,
		ClientKey: clientKey,
	}

	components.СonvertAssign(&wsBroadcast.Message, message)

	return wsBroadcast
}
