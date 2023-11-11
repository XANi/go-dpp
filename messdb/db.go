package messdb

import (
	"github.com/XANi/go-dpp/mq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)
import "gorm.io/driver/sqlite"

type Config struct {
	Path   string
	MQ     *mq.MQ
	Logger *zap.SugaredLogger
}

type MessDB struct {
	db *gorm.DB
	mq *mq.MQ
	l  *zap.SugaredLogger
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
		db: db,
		mq: cfg.MQ,
		l:  cfg.Logger,
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
