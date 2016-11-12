package main

import (
	"github.com/op/go-logging"
	"goji.io"
	"goji.io/pat"
	"os"
	//	"golang.org/x/net/context"
	"./config"
	"./overlord"
	"./web"
	"github.com/XANi/go-dpp/puppet"
	"github.com/XANi/go-yamlcfg"
	"net/http"
	"time"
)

var version string
var log = logging.MustGetLogger("main")
var stdout_log_format = logging.MustStringFormatter("%{color:bold}%{time:2006-01-02T15:04:05.9999Z-07:00}%{color:reset}%{color} [%{level:.1s}] %{color:reset}%{shortpkg}[%{longfunc}] %{message}")
var listenAddr = "127.0.0.1:3002"
var exit = make(chan bool)

func main() {
	stderrBackend := logging.NewLogBackend(os.Stderr, "", 0)
	stderrFormatter := logging.NewBackendFormatter(stderrBackend, stdout_log_format)
	logging.SetBackend(stderrFormatter)
	logging.SetFormatter(stdout_log_format)

	log.Info("Starting app")
	log.Debugf("version: %s", version)
	cfgFiles := []string{
		"./cfg/dpp.conf",
		"./cfg/dpp.default.conf",
		"/etc/my/cnf.yaml",
	}
	var cfg config.Config
	err := yamlcfg.LoadConfig(cfgFiles, &cfg)
	if err != nil {
		log.Errorf("Config error: %+v", err)
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
	time.Sleep(100 * time.Millisecond)
	pup.Run()
	e := <-exit
	_ = e
}
