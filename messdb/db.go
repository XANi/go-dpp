package messdb

import (
	"github.com/XANi/go-dpp/mq"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)
import "gorm.io/driver/sqlite"

type Config struct {
	Node   string
	Path   string
	MQ     *mq.MQ
	Logger *zap.SugaredLogger
}

type MessDB struct {
	node string
	db   *gorm.DB
	mq   *mq.MQ
	l    *zap.SugaredLogger
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
		db:   db,
		mq:   cfg.MQ,
		l:    cfg.Logger,
		node: cfg.Node,
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
	return nil
}

func (m *MessDB) Set(key string, value []byte, expires ...time.Duration) error {
	r := KV{
		Key:   key,
		Owner: m.node,
		Value: value,
	}
	q := m.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&r)
	return q.Error
}

func (m *MessDB) Get(key string) (value []byte, found bool, err error) {
	r := KV{}
	q := m.db.Limit(1).Find(&r, "key = ?", key)
	if q.RowsAffected < 1 {
		return value, true, nil
	}
	if q.Error != nil {
		return value, false, q.Error
	}
	return r.Value, true, nil
}
