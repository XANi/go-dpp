package mq

import (
	"github.com/zerosvc/go-zerosvc"
	"time"
)

type Config struct {
	Address string `yaml:"address"`

}


type MQ struct {
	Info map[string]interface{}
}

func New(nodeName string, cfg Config) *MQ {
	tr := zerosvc.NewTransport(zerosvc.TransportMQTT,cfg.Address,zerosvc.TransportMQTTConfig{})
	node := zerosvc.NewNode(nodeName)
	tr.Connect()
	node.SetTransport(tr)
	e := zerosvc.Event{
		ReplyTo:     "",
		Redelivered: false,
		NeedsAck:    false,
		Headers: nil,
		Body:        []byte("heartbeat"),
	}
	e.Prepare()
	go func() {
		node.SendEvent("dpp/heartbeat/"+nodeName, e)
		time.Sleep(time.Minute*10)
	}()
	return &MQ{}
}