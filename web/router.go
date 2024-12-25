package web

import (
	"fmt"
	"github.com/XANi/go-dpp/messdb"
	"github.com/XANi/go-dpp/puppet"
	"github.com/efigence/go-mon"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"
)

type WebBackend struct {
	l   *zap.SugaredLogger
	al  *zap.SugaredLogger
	r   *gin.Engine
	cfg *Config
	db  *messdb.MessDB
}

type Config struct {
	Logger            *zap.SugaredLogger                                               `yaml:"-"`
	AccessLogger      *zap.SugaredLogger                                               `yaml:"-"`
	ListenAddr        string                                                           `yaml:"listen_addr"`
	UnixSocketDir     string                                                           `yaml:"-"`
	Version           string                                                           `yaml:"-"`
	DB                *messdb.MessDB                                                   `yaml:"-"`
	LastRunStatusFunc func() (success bool, stats puppet.LastRunSummary, ts time.Time) `yaml:"-"`
}

func New(cfg Config, webFS fs.FS) (backend *WebBackend, err error) {
	if cfg.Logger == nil {
		panic("missing logger")
	}
	if len(cfg.ListenAddr) == 0 {
		panic("missing listen addr")
	}
	w := WebBackend{
		l:   cfg.Logger,
		al:  cfg.AccessLogger,
		cfg: &cfg,
		db:  cfg.DB,
	}
	if cfg.AccessLogger == nil {
		w.al = w.l //.Named("accesslog")
	}
	r := gin.New()
	w.r = r
	gin.SetMode(gin.ReleaseMode)
	t, err := template.ParseFS(webFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("error loading templates: %s", err)
	}
	r.SetHTMLTemplate(t)
	// for zap logging
	r.Use(ginzap.GinzapWithConfig(w.al.Desugar(), &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        false,
		SkipPaths: []string{
			"/_status/health",
			"/_status/metrics",
			"/last_run",
		},
	}))
	//r.Use(ginzap.RecoveryWithZap(w.al.Desugar(), true))
	// basic logging to stdout
	//r.Use(gin.LoggerWithWriter(os.Stdout))
	r.Use(gin.Recovery())

	// monitoring endpoints
	r.GET("/_status/health", gin.WrapF(mon.HandleHealthcheck))
	r.HEAD("/_status/health", gin.WrapF(mon.HandleHealthcheck))
	r.GET("/_status/metrics", gin.WrapF(mon.HandleMetrics))
	defer mon.GlobalStatus.Update(mon.StatusOk, "ok")
	// healthcheckHandler, haproxyStatus := mon.HandleHealthchecksHaproxy()
	// r.GET("/_status/metrics", gin.WrapF(healthcheckHandler))
	w.addMessDBAPI()
	httpFS := http.FileServer(http.FS(webFS))
	r.GET("/s/*filepath", func(c *gin.Context) {
		// content is embedded under static/ dir
		p := strings.Replace(c.Request.URL.Path, "/s/", "/static/", -1)
		c.Request.URL.Path = p
		//c.Header("Cache-Control", "public, max-age=3600, immutable")
		httpFS.ServeHTTP(c.Writer, c.Request)
	})
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": c.Request.RemoteAddr,
		})
	})
	r.GET("/version", func(c *gin.Context) {
		c.String(http.StatusOK, cfg.Version)
	})
	r.GET("/last_run", w.Status)
	r.HEAD("/last_run", w.Status)
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.tmpl", gin.H{
			"notfound": c.Request.URL.Path,
		})
	})

	return &w, nil
}

func (b *WebBackend) Run() error {
	b.l.Infof("listening on %s", b.cfg.ListenAddr)
	if b.cfg.UnixSocketDir != "" {
		filename := b.cfg.UnixSocketDir + "/dpp.socket"
		go func() {
			st, err := os.Stat(b.cfg.UnixSocketDir)
			if err != nil {
				err = os.Mkdir(b.cfg.UnixSocketDir, 0700)
				if err != nil {
					b.l.Errorf("could not create %s %s", b.cfg.UnixSocketDir, err)
					return
				}
			} else {
				if !st.IsDir() {
					b.l.Errorf("%s is not a directory, not starting socket", b.cfg.UnixSocketDir)
					return
				}
			}
			b.l.Infof("running on unix socket %s", filename)
			// https://github.com/golang/go/issues/70985
			if _, err := os.Stat(filename); err == nil {
				os.Remove(filename)
			}
			b.l.Errorf("failed starting unix socket: %s", b.r.RunUnix(filename))
		}()
	}
	return b.r.Run(b.cfg.ListenAddr)
}

func (b *WebBackend) Status(c *gin.Context) {
	ok, stats, ts := b.cfg.LastRunStatusFunc()
	status := http.StatusInternalServerError
	if ok {
		status = http.StatusOK
	}
	if !ts.IsZero() {
		c.String(status, fmt.Sprintf("last: %s, changes: %v, dpp version %s",
			time.Now().Sub(ts).Round(time.Second),
			stats.Events,
			b.cfg.Version,
		))
	} else {
		c.String(http.StatusOK, "waiting for first run [%s]", b.cfg.Version)
	}
}
