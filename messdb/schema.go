package messdb

import "time"

type KV struct {
	Key       string `gorm:"primaryKey;unique"`
	CreatedAt time.Time
	UpdatedAt time.Time
	LastRead  time.Time
	SyncedAt  time.Time
	Owner     string
	Value     []byte
}
