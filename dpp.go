package main

import (
	"github.com/op/go-logging"
	"github.com/urfave/cli"
	"os"
	"os/signal"
	"syscall"
	//	"golang.org/x/net/context"
	"github.com/XANi/go-dpp/config"
	"github.com/XANi/go-dpp/deploy"
	"github.com/XANi/go-dpp/overlord"
	"github.com/XANi/go-dpp/web"
	"github.com/XANi/go-yamlcfg"
	"time"
)

var version string
var log = logging.MustGetLogger("main")
var stdout_log_format = logging.MustStringFormatter("%{color:bold}%{time:2006-01-02T15:04:05}%{color:reset}%{color} [%{level:.1s}] %{shortpkg}%{color:reset} %{message}")
var stdout_debug_log_format = logging.MustStringFormatter("%{color:bold}%{time:2006-01-02T15:04:05.99Z-07:00}%{color:reset}%{color} [%{level:.1s}] %{color:reset}%{shortpkg}[%{longfunc}] %{message}")
var stderrBackend = logging.NewLogBackend(os.Stderr, "", 0)

var exit = make(chan bool)
var runPuppet = make(chan bool, 1)

func main() {
	stderrFormatter := logging.NewBackendFormatter(stderrBackend, stdout_log_format)
	logging.SetBackend(stderrFormatter)
	app := cli.NewApp()
	app.Name = "DPP"
	app.Usage = "Distributed puppet runner"
	app.Version = version
	app.Action = func(c *cli.Context) error {
		MainLoop()
		os.Exit(0)
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:    "package",
			Aliases: []string{"p"},
			Usage:   "prepare deploy package",
			Action: func(c *cli.Context) error {
				out := `/tmp/dpp.tar.gz`
				log.Noticef("Preparing deploy package in %s", out)
				d, err := deploy.NewDeployer(deploy.Config{})
				if err != nil {
					log.Errorf("deploy config error: %s", err)
					os.Exit(1)
				}
				err = d.PrepareDeployPackage(out)
				if err != nil {
					log.Errorf("packaging error: %s", err)
					os.Exit(1)
				}
				log.Noticef("deploy prepared")
				os.Exit(0)
				return nil
			},
		},
	}
	app.Run(os.Args)

}

func MainLoop() {
	log.Debugf("version: %s", version)
	cfgFiles := []string{
		"$HOME/.config/dpp/cnf.yaml",
		"/etc/dpp/config.yaml",
		"./cfg/dpp.conf",
		"./cfg/dpp.default.conf",
	}
	cfg := config.Config{
		RepoPollInterval: 600,
		WorkDir:          "/var/lib/dpp",
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
		os.Exit(1)
	}
	if len(cfg.RepoDir) < 1 {
		cfg.RepoDir = cfg.WorkDir + "/repos"
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
	web.Listen()
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
		r.Run()
		for {
			select {
			case <-runPuppet:
				r.Run()
			case <-time.After(time.Second * time.Duration(cfg.Puppet.ScheduleRun)):
				r.Run()
			}
		}
	}()
	signalUSR1 := make(chan os.Signal, 1)
	signalUSR2 := make(chan os.Signal, 1)
	signal.Notify(signalUSR1, syscall.SIGUSR1)
	signal.Notify(signalUSR2, syscall.SIGUSR2)
	go func() {
		for range signalUSR1 {
			log.Notice("Got SIGUSR1, queuing puppet run")
			runPuppet <- true
		}
	}()
	go func() {
		for range signalUSR2 {
			log.Notice("Got SIGUSR2, queuing repo update")
			r.Update()
		}
	}()
	e := <-exit
	_ = e
}

// deployOut := `/tmp/dpp.tar.gz`
// log.Noticef("Preparing deploy package in %s", deployOut)
// d, errd := deploy.NewDeployer(deploy.Config{})
// log.Warningf("deploy prepared: %s | %s", d.PrepareDeployPackage(deployOut), errd)
