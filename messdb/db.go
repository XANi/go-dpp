package messdb

import (
	"github.com/XANi/go-dpp/mq"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sync/atomic"
	"time"
)

type Config struct {
	Node   string
	Path   string
	MQ     *mq.MQ
	Logger *zap.SugaredLogger
}

type MessDB struct {
	node      string
	db        *gorm.DB
	mq        *mq.MQ
	l         *zap.SugaredLogger
	dropCtr   atomic.Uint64
	sendQueue chan *KV
}

func New(cfg Config) (*MessDB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&KV{})
	if err != nil {
		return nil, err
	}
	mdb := &MessDB{
		db:        db,
		mq:        cfg.MQ,
		l:         cfg.Logger,
		node:      cfg.Node,
		sendQueue: make(chan *KV, 256),
	}
	err = db.AutoMigrate(&KV{})
	if err != nil {
		return nil, err
	}
	return mdb, mdb.startSync()
}

func (m *MessDB) startSync() error {
	ev, err := m.mq.Node.GetEventsCh("dpp/messdb/#")
	if err != nil {
		return err
	}
	_ = ev
	go func() {
		for ev := range m.sendQueue {
			e := m.mq.Node.NewEvent()
			e.Marshal(ev)
			m.mq.Node.SendEvent("dpp/messdb/"+m.node, e)
		}
	}()
	go func() {
		for {
			records := []KV{}
			m.db.Limit(100).Find(&records, "synced_at < ?", time.Now().Add(time.Hour*-4))
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
	return nil
}

func (m *MessDB) Set(key string, value string, expires ...time.Duration) error {
	r := KV{
		Key:   key,
		Owner: m.node,
		Value: value,
	}
	q := m.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&r)
	if q.Error == nil {
		// TODO send only changes
		select {
		case m.sendQueue <- &r:
		default:
			m.dropCtr.Add(1)
		}
	}
	return q.Error
}

func (m *MessDB) Get(key string) (value string, found bool, err error) {
	r := KV{}
	q := m.db.Limit(1).Find(&r, "key = ?", key)
	if q.RowsAffected < 1 {
		return value, false, nil
	}
	if q.Error != nil {
		return value, false, q.Error
	}
	return r.Value, true, nil
}

func (m *MessDB) validateKey(key string) error {
	return nil
}
