package mq

import (
	"github.com/XANi/go-dpp/common"
	"github.com/zerosvc/go-zerosvc"
	"time"
)

type Config struct {
	Address string `yaml:"address"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`


}


type MQ struct {
	Info map[string]interface{}
	Node *zerosvc.Node
}

func New(cfg Config, runtime common.Runtime) (*MQ, error) {
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = time.Minute * 10
	}
	nodeName := zerosvc.GetFQDN() + "@dpp"
	node,err := zerosvc.New(zerosvc.Config{
		NodeName:     nodeName,
		Transport:     zerosvc.MQTTTransport(cfg.Address,zerosvc.TransportMQTTConfig{}),
		AutoHeartbeat: true,
		AutoSigner: func(new []byte) (old []byte) { return []byte{}},
	})
	if err != nil {
		return nil,err
	}
	var mq MQ
	mq.Node = node
	return &mq,nil
}