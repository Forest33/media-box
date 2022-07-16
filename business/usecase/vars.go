package usecase

import (
	"media-box-ng/adapter/broker"
)

type Broker interface {
	Start() error
	PublishState(data []byte)
	Subscribe(topic string, handler broker.MessageHandler)
	SetConnectHandler(h broker.ConnectHandler)
	SetDisconnectHandler(h broker.DisconnectHandler)
}

type Image interface {
	Get(url string) (string, error)
}

var (
	stateTopic string
)

func SetStateTopic(t string) {
	stateTopic = t
}
