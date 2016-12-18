package main

import (
	"github.com/op/go-logging"
	"os"
	//	"golang.org/x/net/context"
	"github.com/XANi/go-dpp/config"
	"github.com/XANi/go-dpp/overlord"
	"github.com/XANi/go-dpp/puppet"
	"github.com/XANi/go-dpp/web"
	"github.com/XANi/go-yamlcfg"
	"time"
)

var version string
var log = logging.MustGetLogger("main")
var stdout_log_format = logging.MustStringFormatter("%{color:bold}%{time:2006-01-02T15:04:05}%{color:reset}%{color} [%{level:.1s}] %{shortpkg}%{color:reset} %{message}")
var stdout_debug_log_format = logging.MustStringFormatter("%{color:bold}%{time:2006-01-02T15:04:05.99Z-07:00}%{color:reset}%{color} [%{level:.1s}] %{color:reset}%{shortpkg}[%{longfunc}] %{message}")

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
		ListenAddr:       "127.0.0.1:3002",
		Puppet: config.PuppetInterval{
			StartWait:       60,
			ScheduleRun:     3600,
			MinimumInterval: 300,
		},
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
	web, err := web.New(&cfg)
	if err != nil {
		log.Errorf("starting web server failed with: %s", err)
	}

	log.Infof("Listening on %s", cfg.ListenAddr)
	go web.Listen()

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
	if err != nil {
		log.Panicf("Error while starting puppet: %s", err)
	}
	r, err := overlord.New(&cfg)
	if err != nil {
		log.Panicf("Error while starting overlord: %s", err)
	}
	go func() {
		for {
			log.Noticef("updating repos")
			r.Update()
			time.Sleep(time.Second * time.Duration(cfg.RepoPollInterval))
		}
	}()
	go func() {
		for {
			r.Lock()
			pup.Run()
			r.Unlock()
			time.Sleep(time.Second * time.Duration(cfg.Puppet.ScheduleRun))
		}
	}()
	e := <-exit
	_ = e
}
