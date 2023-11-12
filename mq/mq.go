package mq

import (
	"fmt"
	"github.com/XANi/go-dpp/common"
	uuid "github.com/satori/go.uuid"
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
		// we add a bit of randomness here so running multiple copies of on same node doesn't cause disconnects
		// as MQTT is supposed to disconnect clients with same clientid
		ID:      nodeName + uuid.NewV4().String()[0:8],
		MQTTURL: []*url.URL{addr},
		Logger:  runtime.Logger.Named("transport"),
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
		Logger:        runtime.Logger.Named("node"),
	})
	if err != nil {
		return nil, err
	}
	var mq MQ
	mq.Node = node
	mq.l = cfg.Logger
	return &mq, nil
}
