package overlord

import (
	"fmt"
	"github.com/XANi/go-dpp/config"
	"github.com/XANi/go-dpp/puppet"
	"github.com/XANi/go-dpp/repo"
	"github.com/efigence/go-mon"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Overlord struct {
	l              *zap.SugaredLogger
	cfg            *config.Config
	repos          map[string]*repo.Repo
	puppet         *puppet.Puppet
	repoUpdateLock sync.WaitGroup
	sync.Mutex
}

func New(cfg *config.Config) (*Overlord, error) {
	var overlord Overlord
	var err error
	repoPath := make(map[string]string, len(cfg.UseRepos))
	overlord.cfg = cfg
	overlord.repos = make(map[string]*repo.Repo)
	overlord.puppet, err = initPuppet(cfg)
	overlord.l = cfg.Logger
	if err != nil {
		return nil, err
	}
	for _, repoName := range cfg.UseRepos {
		if _, ok := cfg.Repo[repoName]; !ok {
			cfg.Logger.Errorf("Repo %s specified to use but there is no definition of it! Skipping", repoName)
			if cfg.KillOnBadConfig {
				log.Panicf("incorrect config, failing")
			}
		}
		repoPath[repoName] = cfg.RepoDir + "/" + repoName
		repoCfg := repo.Config{
			PullAddress: cfg.Repo[repoName].PullUrl,
			Branch:      cfg.Repo[repoName].Branch,
			TargetDir:   repoPath[repoName],
			GpgKeys:     cfg.Repo[repoName].GpgKeys,
			Debug:       cfg.Repo[repoName].Debug,
			Logger:      overlord.l.Named("repo-" + repoName),
		}
		overlord.repos[repoName], err = repo.New(repoCfg)
		if err != nil {
			overlord.l.Errorf("Error configuring repo [%s], running cleanup", repoName)
			if strings.HasPrefix(repoCfg.TargetDir, "/var/lib/dpp") {
				err := os.RemoveAll(repoCfg.TargetDir)
				if err != nil {
					overlord.l.Panicf("couldn't cleanup dir [%s]: %s", repoCfg.TargetDir, err)
				}
			} else {
				overlord.l.Errorf("refusing to remove dir not in default [/var/lib/dpp] workdir path for safety reasons")
			}
			return nil, err
		}
	}
	return &overlord, err
}

func initPuppet(cfg *config.Config) (*puppet.Puppet, error) {
	modulePath := make([]string, len(cfg.UseRepos))
	repoPath := make(map[string]string, len(cfg.UseRepos))
	for i, k := range cfg.UseRepos {
		modulePath[i] = cfg.RepoDir + "/" + k + "/modules"
		repoPath[k] = cfg.RepoDir + "/" + k
	}
	if len(cfg.ExtraModulePath) > 0 {
		for _, r := range cfg.ExtraModulePath {
			modulePath = append(modulePath, r)
		}
	}
	cfg.Logger.Debugf("Puppet module path: %+v", modulePath)
	if _, err := os.Stat("/etc/facter/facts.d"); os.IsNotExist(err) {
		cfg.Logger.Info("Creating /etc/facter/facts.d")
		err := os.MkdirAll("/etc/facter/facts.d", 0700)
		if err != nil {
			cfg.Logger.Errorf("Error while creating /etc/facter/facts.d: %s", err)
		}
	}
	cfg.Logger.Debug("creating fact puppet_basemodulepath with current module path")
	path := []byte("puppet_basemodulepath=" + strings.Join(modulePath, ":") + "\n")
	err := ioutil.WriteFile("/etc/facter/facts.d/puppet_basemodulepath.txt", path, 0644)
	if err != nil {
		cfg.Logger.Errorf("can't create fact fil for basemodulepath: %s", err)
	}
	return puppet.New(cfg.Logger.Named("puppet"), modulePath, cfg.RepoDir+"/"+cfg.ManifestFrom+"/manifests/")
}

func (o *Overlord) Run() {
	lockfilePath := o.cfg.WorkDir + "/puppet.lock"
	lockfile, err := os.OpenFile(lockfilePath, os.O_APPEND+os.O_CREATE, 0600)
	if err == nil {
		err := syscall.Flock(int(lockfile.Fd()), syscall.LOCK_EX+syscall.LOCK_NB)
		if err == nil {
			err := o.puppet.Run()
			if err != nil {
				o.l.Errorf("err running puppet: %s", err)
			}
			success, summary, ts := o.puppet.LastRunStats()
			if !success {
				mon.GlobalStatus.Update(mon.StateCritical, "puppet run failed")
			} else if v, ok := summary.Resources["failure"]; ok == true && v > 0 {
				mon.GlobalStatus.Update(mon.StateWarning, fmt.Sprintf("failed resources: %d", v))
			} else if v, ok := summary.Events["failure"]; ok == true && v > 0 {
				mon.GlobalStatus.Update(mon.StateWarning, fmt.Sprintf("failed resources: %d", v))
			} else {
				mon.GlobalStatus.Update(mon.StateOk, fmt.Sprintf("last puppet run %s", ts.Format("2006-01-02 15:04")))
			}
		} else {
			o.l.Errorf("Puppet run already in progress [lockfile: %s]", lockfilePath)
		}
		lockfile.Close()
	} else {
		o.l.Errorf("Can't open lock %s: %s, running anyway", lockfilePath, err)
		o.puppet.Run()
	}
}

func (o *Overlord) State() (success bool, lastRunSummary puppet.LastRunSummary, ts time.Time) {
	return o.puppet.LastRunStats()
}
func (o *Overlord) Update() error {
	var wg sync.WaitGroup
	o.Lock()
	for name, r := range o.repos {
		go func() {}()
		o.l.Debugf("Updating repo %s", name)
		wg.Add(1)
		go func(r *repo.Repo, wg *sync.WaitGroup) {
			err := r.Update()
			if err != nil {
				o.l.Warnf("Error updating %s: %s", name, err)
			}
			wg.Done()
		}(r, &wg)
	}
	wg.Wait()
	o.l.Debug("update done")
	o.Unlock()
	return nil
}

func (o *Overlord) LastRunSummary() (success bool, stats puppet.LastRunSummary, ts time.Time) {
	return o.puppet.LastRunStats()

}
