package repo

import (
	"github.com/XANi/go-gitcli"
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
	Force bool
}
func New(cfg Config) (r *Repo, err error) {
	var repo Repo
	gitRepo := gitcli.New(cfg.TargetDir)
	err = 	gitRepo.Init()
	if err != nil {return r, err}
	err = gitRepo.SetRemote("origin", cfg.PullAddress)
	if err != nil {return r, err}
	err = gitRepo.Fetch("origin")
	if err != nil {return r, err}
	repo.repo = &gitRepo
	return &repo,err
}
