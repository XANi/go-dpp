package messdb

import (
	"fmt"
	"github.com/XANi/go-dpp/mq"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

type Config struct {
	Node         string
	Path         string
	MQ           *mq.MQ
	SharedPrefix string
	Logger       *zap.SugaredLogger
}

type MessDB struct {
	node            string
	db              *gorm.DB
	mq              *mq.MQ
	l               *zap.SugaredLogger
	dropCtr         atomic.Uint64
	sharedPrefix    string
	sendQueue       chan *KV
	readUpdateQueue chan string
}

func New(cfg Config) (*MessDB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{})
	db.Exec("PRAGMA  journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&KV{})
	if err != nil {
		return nil, err
	}
	if len(cfg.SharedPrefix) == 0 {
		cfg.SharedPrefix = "shared::"
	}
	mdb := &MessDB{
		db:              db,
		mq:              cfg.MQ,
		l:               cfg.Logger,
		node:            cfg.Node,
		sharedPrefix:    cfg.SharedPrefix,
		sendQueue:       make(chan *KV, 256),
		readUpdateQueue: make(chan string, 256),
	}
	err = db.AutoMigrate(&KV{})
	if err != nil {
		return nil, err
	}
	return mdb, mdb.startSync()
}

func (m *MessDB) startSync() error {
	incoming, err := m.mq.Node.GetEventsCh("messdb/#")
	if err != nil {
		return err
	}
	go func() {
		for ev := range m.sendQueue {
			e := m.mq.Node.NewEvent()
			e.Marshal(ev)
			m.mq.Node.SendEvent("messdb/"+m.node, e)
		}
	}()
	go func() {
		for {
			records := []KV{}
			m.db.Limit(100).Find(&records, "synced_at < ? AND key LIKE 'shared::%'", time.Now().Add(time.Hour*-4))
			if len(records) == 0 {
				time.Sleep(time.Minute * 120)
				continue
			}
			m.l.Infof("updating %d records", len(records))
			for _, r := range records {
				// we are fine waiting here, no need for select, if queue is down so is sync...
				m.sendQueue <- &r
				m.db.Model(&r).Where("key = ?", r.Key).Update("synced_at", time.Now())
			}
			if len(records) < 100 {
				time.Sleep(time.Minute * 120)
			} else {
				time.Sleep(time.Minute * 30)
			}
		}
	}()
	go func() {
		for ev := range incoming {
			key := KV{}
			err := ev.Unmarshal(&key)
			if err != nil {
				m.l.Errorf("error unmarshalling incoming event:[%s] %+v", err, ev)
				continue
			}
			search := KV{
				Key: key.Key,
			}
			tx := m.db.Find(&search).First(&search)
			if tx.Error != nil {
				if tx.Error == gorm.ErrRecordNotFound {
					m.db.Save(&key)
				} else {
					m.l.Errorf("Error looking for key [%s]: %s", key.Key, err)
				}
				continue
			}
			if key.Owner != search.Owner {
				m.l.Warnf("tried to set same key from multiple hosts[%s: %s %s], ignoring", key.Key, key.Owner, search.Owner)
			} else if search.UpdatedAt.Before(key.UpdatedAt) {
				m.db.Save(&key)
			}
		}
	}()
	go func() {
		time.Sleep(time.Second * 10)
		for k := range m.readUpdateQueue {
			m.l.Infof("updating read on %s", k)
			tx := m.db.Model(&KV{}).Where("key = ?", k).Update("last_read", time.Now())
			if tx.Error != nil {
				m.l.Errorf("error updating %s: %s", k, err)
			}
		}
	}()
	return nil
}

func (m *MessDB) Set(key string, value []byte, expires ...time.Duration) error {
	r := KV{
		Key:   key,
		Owner: m.node,
		Value: value,
	}
	search := &KV{}
	tx := m.db.Find(&KV{Key: key}).First(&search)
	if tx.Error == nil && search.Owner != r.Owner {
		return fmt.Errorf("key %s already has different owner: %s", key, search.Owner)
	}
	fmt.Printf("-- %+v --\n", search)
	q := m.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&r)
	if q.Error == nil && strings.HasPrefix(key, m.sharedPrefix) {
		// TODO send only changes
		select {
		case m.sendQueue <- &r:
		default:
			m.dropCtr.Add(1)
		}
	}
	return q.Error
}

func (m *MessDB) Get(key string) (value []byte, found bool, err error) {
	r := KV{}
	q := m.db.Limit(1).Find(&r, "key = ?", key)
	if q.RowsAffected < 1 {
		return value, false, nil
	}
	if q.Error != nil {
		return value, false, q.Error
	}
	if time.Now().Sub(r.LastRead) > time.Duration(int64(time.Hour)*24+rand.Int63n(int64(time.Hour)*24)) {
		select {
		case m.readUpdateQueue <- key:
			m.l.Infof("updating read time on %s", key)
		default:
		}
	}
	return r.Value, true, nil
}

func (m *MessDB) validateKey(key string) error {
	return nil
}
