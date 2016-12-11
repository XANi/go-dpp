package repo

import (
	"github.com/XANi/go-gitcli"
//	"github.com/op/go-gpgcli"
)



type Repo struct {
	branch string
	address string
	dir string
	force bool
	gpgKeys []string
	repo *gitcli.Repo
}

type Config struct {
	PullAddress string
	Branch string
	TargetDir string
	GpgKeys []string
	Force bool
}
func New(cfg Config) (r *Repo, err error) {
	var repo Repo
	// 		if len(cfg.Repo[repoName].GpgKeys) > 0 {
	// 		repoCfg.GpgKeys
	// 		for _, key := range cfg.Repo[repoName].GpgKeys {
	// 			fingerprint,err := gpg.GetFingerprint(key)
	// 			if err != nil {
	// 				log.Warningf("Couln't resolve gpg fingerprint for key %s in local gpg db: %s", key, err)
	// 				if cfg.KillOnBadConfig {
	// 					log.Panicf("incorrect config, failing")
	// 				}
	// 			}
	// 			repoCfg.Gpg


	// 		}
	// 	}
	// 	overlord.repos[name] = repo.New(repo.Config{
	gitRepo := gitcli.New(cfg.TargetDir)
	err = 	gitRepo.Init()
	if err != nil {return r, err}
	err = gitRepo.SetRemote("origin", cfg.PullAddress)
	if err != nil {return r, err}
	err = gitRepo.Fetch("origin")
	if err != nil {return r, err}
	err = gitRepo.Checkout("remotes/origin/master")
	if err != nil {return r, err}
	err = gitRepo.SubmoduleInit()
	if err != nil {return r, err}
	err = gitRepo.SubmoduleSync()
	if err != nil {return r, err}
	err = gitRepo.SubmoduleUpdate()
	if err != nil {return r, err}
	repo.repo = &gitRepo
	return &repo,err
}
