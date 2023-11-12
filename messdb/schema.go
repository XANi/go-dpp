package messdb

import "time"

type KV struct {
	Key       string `gorm:"primaryKey;unique"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Owner     string
	Value     []byte
}
