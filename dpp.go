package main

import (
	"github.com/op/go-logging"
	"goji.io"
	"goji.io/pat"
	"os"
	//	"golang.org/x/net/context"
	"github.com/XANi/go-dpp/config"
	"github.com/XANi/go-dpp/overlord"
	"github.com/XANi/go-dpp/web"
	"github.com/XANi/go-dpp/puppet"
	"github.com/XANi/go-yamlcfg"
	"net/http"
	"time"
)

var version string
var log = logging.MustGetLogger("main")
var stdout_log_format = logging.MustStringFormatter("%{color:bold}%{time:2006-01-02T15:04:05}%{color:reset}%{color} [%{level:.1s}] %{shortpkg}%{color:reset} %{message}")
var stdout_debug_log_format = logging.MustStringFormatter("%{color:bold}%{time:2006-01-02T15:04:05.99Z-07:00}%{color:reset}%{color} [%{level:.1s}] %{color:reset}%{shortpkg}[%{longfunc}] %{message}")
var listenAddr = "127.0.0.1:3002"
var exit = make(chan bool)

func main() {
	stderrBackend := logging.NewLogBackend(os.Stderr, "", 0)
	stderrFormatter := logging.NewBackendFormatter(stderrBackend, stdout_log_format)
	logging.SetBackend(stderrFormatter)

	log.Info("Starting app")
	log.Debugf("version: %s", version)
	cfgFiles := []string{
		"$HOME/.config/dpp/cnf.yaml",
		"/etc/dpp/config.yaml",
		"./cfg/dpp.conf",
		"./cfg/dpp.default.conf",
	}
	cfg := config.Config{
		RepoPollInterval: 600,
	}
	err := yamlcfg.LoadConfig(cfgFiles, &cfg)
	if err != nil {
		log.Errorf("Config error: %+v", err)
	}
	if cfg.Debug {
		stderrFormatter := logging.NewBackendFormatter(stderrBackend, stdout_debug_log_format)
		logging.SetBackend(stderrFormatter)
	}
	log.Debugf("Config: %+v", cfg)
	renderer, err := web.New()
	if err != nil {
		log.Errorf("Renderer failed with: %s", err)
	}
	mux := goji.NewMux()
	mux.Handle(pat.Get("/static/*"), http.StripPrefix("/static", http.FileServer(http.Dir(`public/static`))))
	mux.Handle(pat.Get("/apidoc/*"), http.StripPrefix("/apidoc", http.FileServer(http.Dir(`public/apidoc`))))
	mux.HandleFunc(pat.Get("/"), renderer.HandleRoot)
	log.Infof("Listening on %s", listenAddr)
	go http.ListenAndServe(listenAddr, mux)

	// prepare paths
	log.Infof("%+v", cfg.UseRepos)
	// TODO create parent
	modulePath := make([]string, len(cfg.UseRepos))
	repoPath := make(map[string]string, len(cfg.UseRepos))
	for i, k := range cfg.UseRepos {
		modulePath[i] = cfg.RepoDir + "/" + k + "/puppet/modules"
		repoPath[k] = cfg.RepoDir + "/" + k

	}
	log.Errorf("%+v", modulePath)
	pup, err := puppet.New(modulePath, cfg.RepoDir+"/"+cfg.ManifestFrom+"/puppet/manifests/site.pp")
	log.Info(err)
	r, err := overlord.New(&cfg)
	_ = r
	go func() {
		for {
			time.Sleep(time.Second * time.Duration(cfg.RepoPollInterval))
			log.Noticef("updating")
			r.Update()
		}
	}()
	pup.Run()
	e := <-exit
	_ = e
}
