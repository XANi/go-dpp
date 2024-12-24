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
	"time"
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
	err = db.Set("test", []byte("1234"))
	require.NoError(t, err)

	value, found, err := db.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "1234", string(value))
	assert.True(t, found)

	value, found, err = db.Get("notfound")
	require.NoError(t, err)
	assert.Equal(t, "", string(value))
	assert.False(t, found)

	err = db.Set("test", []byte("2222"))
	require.NoError(t, err)
	value, found, err = db.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "2222", string(value))
	assert.True(t, found)
}

func TestMessDBCluster(t *testing.T) {
	dir := t.TempDir()
	log := zaptest.NewLogger(t).Sugar()
	runtime := common.Runtime{Logger: log}
	cfg := mq.Config{
		Address: getTestMQURL(),
		Logger:  log.Named("mq"),
	}
	node, err := mq.New(cfg, runtime)
	db1, err := New(Config{
		Node:   "test1",
		Path:   dir + "/test1.sqlite",
		MQ:     node,
		Logger: log.Named("messdb1"),
	})
	l := zapgorm2.New(log.Named("db1").Desugar())
	l.SetAsDefault()
	l.LogLevel = logger.Info
	db1.db.Logger = l
	require.NoError(t, err)
	db2, err := New(Config{
		Node:   "test2",
		Path:   dir + "/test2.sqlite",
		MQ:     node,
		Logger: log.Named("messdb2"),
	})
	require.NoError(t, err)
	l = zapgorm2.New(log.Named("db2").Desugar())
	l.SetAsDefault()
	l.LogLevel = logger.Info
	db2.db.Logger = l

	err = db1.Set("shared::test1", []byte("1234"))
	require.NoError(t, err)
	err = db2.Set("shared::test2", []byte("5678"))
	require.NoError(t, err)
	time.Sleep(time.Second)

	value, found, err := db1.Get("shared::test1")
	require.NoError(t, err)
	assert.Equal(t, "1234", string(value))
	assert.True(t, found)

	value, found, err = db1.Get("shared::test2")
	require.NoError(t, err)
	assert.Equal(t, "5678", string(value))
	assert.True(t, found)

	value, found, err = db2.Get("shared::test1")
	require.NoError(t, err)
	assert.Equal(t, "1234", string(value))
	assert.True(t, found)

	value, found, err = db2.Get("shared::test2")
	require.NoError(t, err)
	assert.Equal(t, "5678", string(value))
	assert.True(t, found)

	err = db1.Set("shared::test1", []byte("6666"))
	time.Sleep(time.Second)
	require.NoError(t, err)
	err = db2.Set("shared::test1", []byte("6667"))
	// fail coz the key belongs to db1
	require.Error(t, err)
	value, found, err = db2.Get("shared::test1")
	require.NoError(t, err)
	assert.Equal(t, "6666", string(value))
	assert.True(t, found)
}
