package mq

import (
	"fmt"
	"github.com/XANi/go-dpp/common"
	"github.com/zerosvc/go-zerosvc"
	"time"
)

type Config struct {
	Address           string        `yaml:"address"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
}

type MQ struct {
	Info       map[string]interface{}
	Node       *zerosvc.Node
	leaderName string
	leaderTS   time.Time
}

func New(cfg Config, runtime common.Runtime) (*MQ, error) {
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = time.Minute * 10
	}
	nodeName := zerosvc.GetFQDN() + "@dpp"
	node, err := zerosvc.New(zerosvc.Config{
		NodeName:      nodeName,
		Transport:     zerosvc.MQTTTransport(cfg.Address, zerosvc.TransportMQTTConfig{}),
		AutoHeartbeat: true,
		AutoSigner:    func(new []byte) (old []byte) { return []byte{} },
	})
	if err != nil {
		return nil, err
	}
	var mq MQ
	mq.Node = node
	go mq.masterElection()
	return &mq, nil
}

type Election struct {
	Ts   time.Time
	Node string
}

func (m *MQ) masterElection() {
	for {
		ch, err := m.Node.GetEventsCh("dpp/leader_election")
		if err != nil {
			time.Sleep(time.Minute)
			continue
		}
		for {
			select {
			case ev := <-ch:
				e := Election{}
				err := ev.Unmarshal(&e)
				if err != nil {
					continue
				}
				fmt.Printf("event: %+v\n", ev)
				if m.leaderTS.Before(e.Ts) {
					m.leaderTS = e.Ts.UTC()
					m.leaderName = e.Node
				}
			case <-time.After(time.Minute):
				// if it is old, throw our lot
				if time.Now().UTC().Sub(m.leaderTS) > (time.Minute * 5) {
					leaderEv := m.Node.NewEvent()
					retain := time.Now().Add(time.Minute * 10).UTC()
					leaderEv.RetainTill = &retain

					leaderEv.Marshal(&Election{
						Ts:   time.Now().UTC(),
						Node: m.Node.Name,
					})
					err := leaderEv.Send("dpp/leader_election")
					if err != nil {
						fmt.Printf("err: %s", err)
					}
				} else { //else if we're master, send event every minute
					if m.Node.Name == m.leaderName {
						leaderEv := m.Node.NewEvent()
						leaderEv.Marshal(&Election{
							Ts:   time.Now().UTC(),
							Node: m.Node.Name,
						})
						retain := time.Now().Add(time.Minute * 10).UTC()
						leaderEv.RetainTill = &retain
						leaderEv.Send("dpp/leader_election")
					}
				}
			}
		}
	}
}
