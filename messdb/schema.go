package messdb

import "time"

type KV struct {
	Key       string `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Owner     string
	Value     string
}
