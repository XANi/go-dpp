package overlord

import (
	"github.com/op/go-logging"
	"github.com/XANi/go-dpp/config"
	"github.com/XANi/go-dpp/puppet"
	"github.com/XANi/go-dpp/repo"
	"os"
	"syscall"
	"sync"
)

var log = logging.MustGetLogger("main")

type Overlord struct {
	cfg *config.Config
	repos map[string]*repo.Repo
	puppet *puppet.Puppet
	repoUpdateLock sync.WaitGroup
	sync.Mutex
}

func New(cfg *config.Config)  (o *Overlord, err error) {
	var overlord Overlord
	modulePath := make([]string, len(cfg.UseRepos))
	repoPath := make(map[string]string, len(cfg.UseRepos))
	overlord.cfg = cfg
	overlord.repos = make(map[string]*repo.Repo)
	overlord.puppet, err = initPuppet(cfg)
	if err != nil { return nil, err}


	for i, repoName := range cfg.UseRepos {
		if _, ok := cfg.Repo[repoName]; !ok {
			log.Errorf("Repo %s specified to use but there is no definition of it! Skipping", repoName)
			if cfg.KillOnBadConfig { log.Panicf("incorrect config, failing") }
		}

		modulePath[i] = cfg.RepoDir + "/" + repoName + "/puppet/modules"
		repoPath[repoName] = cfg.RepoDir + "/" + repoName
		repoCfg := repo.Config{
			PullAddress: cfg.Repo[repoName].PullUrl,
			Branch: cfg.Repo[repoName].Branch,
			TargetDir: repoPath[repoName],
			GpgKeys: cfg.Repo[repoName].GpgKeys,
			Debug: cfg.Repo[repoName].Debug,
		}
		overlord.repos[repoName], err = repo.New(repoCfg)
		if err != nil {
			log.Panicf("Can't configure repo %s: %s", repoName,err)
		}
	}
	return &overlord, err
}

func initPuppet(cfg *config.Config) (*puppet.Puppet, error) {
	modulePath := make([]string, len(cfg.UseRepos))
	repoPath := make(map[string]string, len(cfg.UseRepos))
	for i, k := range cfg.UseRepos {
		modulePath[i] = cfg.RepoDir + "/" + k + "/puppet/modules"
		repoPath[k] = cfg.RepoDir + "/" + k
	}
	log.Debugf("Puppet module path: %+v", modulePath)
	return puppet.New(modulePath, cfg.RepoDir+"/"+cfg.ManifestFrom+"/puppet/manifests/")
}

func (o *Overlord)Run() {
	lockfilePath := o.cfg.WorkDir + "/puppet.lock"
	lockfile, err := os.OpenFile(lockfilePath, os.O_APPEND +  os.O_CREATE, 0600)
	if err == nil {
		err := syscall.Flock(int(lockfile.Fd()), syscall.LOCK_EX + syscall.LOCK_NB)
		if err == nil {
			o.puppet.Run()
		} else {
			log.Errorf("Puppet run already in progress [lockfile: %s]", lockfilePath)
		}
		lockfile.Close()
	} else {
		log.Errorf("Can't open lock %s: %s, running anyway", lockfilePath, err)
		o.puppet.Run()
	}
}

func (o *Overlord)Update() error {
	var wg sync.WaitGroup
	o.Lock()
	for name, r := range o.repos {
		go func() {} ()
		log.Debugf("Updating repo %s",name)
		wg.Add(1)
		go func(r *repo.Repo, wg *sync.WaitGroup) {
			err := r.Update()
			if err != nil {
				log.Warningf("Error updating %s: %s", name, err)
			}
			wg.Done()
		} (r, &wg)
	}
	wg.Wait()
	log.Debug("update done")
	o.Unlock()
	return nil
}
