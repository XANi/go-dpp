package messdb

import (
	"github.com/XANi/go-dpp/common"
	"github.com/XANi/go-dpp/mq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gorm.io/gorm/logger"
	"moul.io/zapgorm2"
	"os"
	"testing"
)

func getTestMQURL() string {
	defaultURL := "tcp://guest:guest@127.0.0.1:1883"
	envUrl := os.Getenv("TEST_MQTT_URL")
	if len(envUrl) > 4 {
		defaultURL = envUrl
	}
	return defaultURL
}
func TestMessDB(t *testing.T) {
	log := zaptest.NewLogger(t).Sugar()
	runtime := common.Runtime{Logger: log}
	cfg := mq.Config{
		Address: getTestMQURL(),
		Logger:  log.Named("mq"),
	}
	node, err := mq.New(cfg, runtime)
	require.NoError(t, err)
	db, err := New(Config{
		Node:   "test",
		Path:   "/tmp/test.sqlite",
		MQ:     node,
		Logger: log.Named("messdb"),
	})
	l := zapgorm2.New(log.Desugar())
	l.SetAsDefault()
	l.LogLevel = logger.Info
	db.db.Logger = l

	//db.db = db.db.Debug()
	require.NoError(t, err)
	err = db.Set("test", "1234")
	require.NoError(t, err)

	value, found, err := db.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "1234", value)
	assert.True(t, found)

	value, found, err = db.Get("notfound")
	require.NoError(t, err)
	assert.Equal(t, "", value)
	assert.False(t, found)

	err = db.Set("test", "2222")
	require.NoError(t, err)
	value, found, err = db.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "2222", value)
	assert.True(t, found)

}
