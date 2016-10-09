package main

import (
	"github.com/op/go-logging"
	"goji.io"
	"goji.io/pat"
	"os"
	"strings"
	//	"golang.org/x/net/context"
	"./web"
	"github.com/XANi/go-dpp/config"
	"github.com/XANi/go-dpp/puppet"
	"github.com/XANi/go-yamlcfg"
	"net/http"
)

var version string
var log = logging.MustGetLogger("main")
var stdout_log_format = logging.MustStringFormatter("%{color:bold}%{time:2006-01-02T15:04:05.9999Z-07:00}%{color:reset}%{color} [%{level:.1s}] %{color:reset}%{shortpkg}[%{longfunc}] %{message}")
var listenAddr = "127.0.0.1:3002"

func main() {
	stderrBackend := logging.NewLogBackend(os.Stderr, "", 0)
	stderrFormatter := logging.NewBackendFormatter(stderrBackend, stdout_log_format)
	logging.SetBackend(stderrFormatter)
	logging.SetFormatter(stdout_log_format)

	log.Info("Starting app")
	log.Debugf("version: %s", version)
	if !strings.ContainsRune(version, '-') {
		log.Warning("once you tag your commit with name your version number will be prettier")
	}
	log.Error("now add some code!")
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
	mux.HandleFuncC(pat.Get("/"), renderer.HandleRoot)
	log.Infof("Listening on %s", listenAddr)
	go http.ListenAndServe(listenAddr, mux)

	// prepare paths
	log.Infof("%+v", cfg.UseRepos)
	modulePath := make([]string, len(cfg.UseRepos))
	for i, k := range cfg.UseRepos {
		modulePath[i] = cfg.RepoDir + "/" + k + "/puppet/modules"
	}
	log.Errorf("%+v", modulePath)
	pup, err := puppet.New(modulePath, cfg.RepoDir+"/"+cfg.ManifestFrom+"/puppet/manifests/site.pp")
	log.Info(err)
	pup.Run()
}
