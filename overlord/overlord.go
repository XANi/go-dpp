package overlord

import (
	"github.com/op/go-logging"
	"../config"
	"../repo"
	"sync"
)

var log = logging.MustGetLogger("main")

type Overlord struct {
	cfg *config.Config
	repos map[string]*repo.Repo
	repoUpdateLock sync.WaitGroup
	sync.Mutex
}

func New(cfg *config.Config)  (o *Overlord, err error) {
	var overlord Overlord
	modulePath := make([]string, len(cfg.UseRepos))
	repoPath := make(map[string]string, len(cfg.UseRepos))
	overlord.cfg = cfg
	overlord.repos = make(map[string]*repo.Repo)


	for i, repoName := range cfg.UseRepos {
		if _, ok := cfg.Repo[repoName]; !ok {
			log.Errorf("Repo %s specified to use but there is no definition of it! Skipping")
		}
		if cfg.KillOnBadConfig {
			log.Panicf("incorrect config, failing")
		}
		modulePath[i] = cfg.RepoDir + "/" + repoName + "/puppet/modules"
		repoPath[repoName] = cfg.RepoDir + "/" + repoName
		repoCfg := repo.Config{
			PullAddress: cfg.Repo[repoName].PullUrl,
			Branch: cfg.Repo[repoName].Branch,
			TargetDir: repoPath[repoName],
			GpgKeys: cfg.Repo[repoName].GpgKeys,
		}
		overlord.repos[repoName], err = repo.New(repoCfg)
		if err != nil {
			log.Panicf("Can't configure repo %s: %s", repoName,err)
		}
	}
	return &overlord, err
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
	o.Unlock()
	return nil
}
