package main

import (
	"embed"
	"github.com/XANi/go-dpp/common"
	"github.com/XANi/go-dpp/messdb"
	"github.com/XANi/go-dpp/mq"
	"github.com/XANi/goneric"
	"github.com/urfave/cli"
	"github.com/zerosvc/go-zerosvc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

var exit = make(chan bool)
var runPuppet = make(chan bool, 1)
var debug = false
var log *zap.SugaredLogger

// /* embeds with all files, just dir/ ignores files starting with _ or .
//
//go:embed static templates
var embeddedWebContent embed.FS
var underSystemd bool

func init() {
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	// naive systemd detection. Drop timestamp if running under it
	// broken in new systemd ;/
	if os.Getenv("INVOCATION_ID") != "" || os.Getenv("JOURNAL_STREAM") != "" {

		consoleEncoderConfig.TimeKey = ""
	}
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
	consoleStderr := zapcore.Lock(os.Stderr)
	_ = consoleStderr
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return (lvl < zapcore.ErrorLevel) != (lvl == zapcore.DebugLevel && !debug)
	})
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, os.Stderr, lowPriority),
		zapcore.NewCore(consoleEncoder, os.Stderr, highPriority),
	)
	logger := zap.New(core)
	if debug {
		logger = logger.WithOptions(
			zap.Development(),
			zap.AddCaller(),
			zap.AddStacktrace(highPriority),
		)
	} else {
		logger = logger.WithOptions(
			zap.AddCaller(),
		)
	}
	log = logger.Sugar()

}

func main() {
	app := cli.NewApp()
	app.Name = "DPP"
	app.Usage = "Distributed puppet runner"
	app.Version = version
	app.Action = func(c *cli.Context) error {
		MainLoop(c)
		os.Exit(0)
		return nil
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "socket-dir",
			Usage:  "specify dir for control socket, [dpp.socket] will be created inside",
			Hidden: false,
			Value:  "",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "package",
			Aliases: []string{"p"},
			Usage:   "prepare deploy package",
			Action: func(c *cli.Context) error {
				out := `/tmp/dpp.tar.gz`
				log.Infof("Preparing deploy package in %s", out)
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
				log.Infof("deploy prepared")
				os.Exit(0)
				return nil
			},
		},
	}
	app.Run(os.Args)

}

func MainLoop(c *cli.Context) {
	log.Infof("version: %s", version)
	cfgFiles := []string{
		"$HOME/.config/dpp/cnf.yaml",
		"/etc/dpp/config.yaml",
		"./cfg/dpp.conf",
		"./cfg/dpp.default.conf",
	}
	cfg := config.Config{
		RepoPollInterval: 600,
		WorkDir:          "/var/lib/dpp",
		Puppet: config.PuppetInterval{
			StartWait:       60,
			ScheduleRun:     3600,
			MinimumInterval: 300,
		},
	}
	cfg.NodeName, _ = os.Hostname()
	cfg.UnixSocketDir = c.String("socket-dir")

	err := yamlcfg.LoadConfig(cfgFiles, &cfg)
	if err != nil {
		log.Errorf("Config error: %+v", err)
		os.Exit(1)
	}
	if len(cfg.WorkDir) < 1 {
		cfg.WorkDir = "/var/lib/dpp"
	}
	if len(cfg.RepoDir) < 1 {
		cfg.RepoDir = cfg.WorkDir + "/repos"
	}
	log.Debugf("Config: %+v", cfg)
	runtime := common.Runtime{Logger: log}
	cfg.MQ.Logger = log.Named("mq")
	cfg.MQ.NodeName = cfg.NodeName
	mq, err := mq.New(cfg.MQ, runtime)
	_ = mq
	if err != nil {
		log.Errorf("mq start failed: %", err)
		go func() {
			log.Infof("will restart daemon in 8 hours and try again")
			time.Sleep(time.Hour * 8)
			exit <- true
		}()
	} else {
		log.Infof("connected to MQ at %s", cfg.MQ)
	}
	db, err := messdb.New(messdb.Config{
		Node:   cfg.NodeName,
		Path:   "/var/lib/dpp/messdb.sql",
		MQ:     mq,
		Logger: log.Named("messdb"),
	})
	if err != nil {
		log.Errorf("error initializing database: %s", err)
	}
	if cfg.Web != nil {
		cfg.Web.DB = db
		cfg.Web.Logger = log
		cfg.Web.UnixSocketDir = cfg.UnixSocketDir
		w, err := web.New(*cfg.Web, embeddedWebContent)
		if err != nil {
			log.Errorf("error setting up web server: %s", err)
		}
		go func() {
			log.Errorf("error listening on web socket %s: %s", cfg.Web.ListenAddr, w.Run())
		}()
	}
	cfg.Logger = log
	r, err := overlord.New(&cfg)
	if err != nil {
		log.Panicf("Error while starting overlord: %s", err)
	}
	go func() {
		for {
			log.Infof("updating repos")
			r.Update()
			time.Sleep(time.Second * time.Duration(cfg.RepoPollInterval))
		}
	}()
	go func() {
		r.Run()
		for {
			success, summary, ts := r.State()
			_ = summary
			mq.Node.Lock()
			mq.Node.Services["puppet"] = zerosvc.Service{
				Ok: success,
				Data: struct {
					TS      time.Time `json:"ts" cbor:"ts"`
					Changes int       `json:"changes" cbor:"changes"`
				}{
					TS: ts,
					Changes: goneric.Sum(goneric.MapToSlice(func(k string, v int) int {
						return v
					}, summary.Changes)...),
				},
			}
			mq.Node.Unlock()
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
			log.Info("Got SIGUSR1, queuing update and puppet run")
			r.Update()
			runPuppet <- true
		}
	}()
	go func() {
		for range signalUSR2 {
			log.Info("Got SIGUSR2, exiting after 5 minutes")
			time.Sleep(time.Minute * 5)
			os.Exit(0)

		}
	}()
	e := <-exit
	_ = e
}

// deployOut := `/tmp/dpp.tar.gz`
// log.Infof("Preparing deploy package in %s", deployOut)
// d, errd := deploy.NewDeployer(deploy.Config{})
// log.Warningf("deploy prepared: %s | %s", d.PrepareDeployPackage(deployOut), errd)
