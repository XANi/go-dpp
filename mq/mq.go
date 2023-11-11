package mq

import (
	"fmt"
	"github.com/XANi/go-dpp/common"
	"github.com/zerosvc/go-zerosvc"
	"go.uber.org/zap"
	"net/url"
	"time"
)

type Config struct {
	Address           string             `yaml:"address"`
	HeartbeatInterval time.Duration      `yaml:"heartbeat_interval"`
	Logger            *zap.SugaredLogger `yaml:"-"`
}

type MQ struct {
	Info       map[string]interface{}
	Node       *zerosvc.Node
	l          *zap.SugaredLogger
	leaderName string
	leaderTS   time.Time
}

func New(cfg Config, runtime common.Runtime) (*MQ, error) {
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = time.Minute * 10
	}
	nodeName := zerosvc.GetFQDN() + "@dpp"
	addr, err := url.Parse(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}
	tr, err := zerosvc.NewTransportMQTTv5(zerosvc.ConfigMQTTv5{
		ID:      nodeName,
		MQTTURL: []*url.URL{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating transport: %w", err)
	}
	node, err := zerosvc.NewNode(zerosvc.Config{
		NodeName:      nodeName,
		Transport:     tr,
		AutoHeartbeat: true,
		AutoSigner:    func(new []byte) (old []byte) { return []byte{} },
		EventRoot:     "dpp",
	})
	if err != nil {
		return nil, err
	}
	var mq MQ
	mq.Node = node
	mq.l = cfg.Logger
	return &mq, nil
}
