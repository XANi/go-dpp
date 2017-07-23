package repo

import (
	"github.com/XANi/go-gitcli"
	"fmt"
	"github.com/XANi/go-gpgcli"
)



type Repo struct {
	branch string
	address string
	dir string
	force bool
	gpg bool
	gpgKeys []string
	repo *gitcli.Repo
}

type Config struct {
	PullAddress string
	Branch string
	TargetDir string
	GpgKeys []string
	Force bool
	Debug bool
}
func New(cfg Config) (r *Repo, err error) {
	var repo Repo
	// resolve fingerprints to their full length
	if len(cfg.GpgKeys) > 0 {
		repo.gpg = true
		repo.gpgKeys = make([]string,len(cfg.GpgKeys))
		gpg, err := gpgcli.New()
		if err != nil {
			return nil, err
		}
		for _, key := range cfg.GpgKeys {
			fingerprint,err := gpg.GetFingerprintById(key)
			if err != nil {
				fmt.Errorf("Couldn't resolve gpg fingerprint for key %s in local gpg db: %s, ignoring", key, err)
			}
			repo.gpgKeys = append(repo.gpgKeys, fingerprint)
		}
	}
	gitRepo := gitcli.New(cfg.TargetDir)
	if cfg.Debug {
		gitRepo.SetDebug(true)
	}
	err = gitRepo.Init()
	if repo.gpg {
		gitRepo.SetTrustedSignatures(repo.gpgKeys)
	}
	if err != nil {return nil, err}
	err = gitRepo.SetRemote("origin", cfg.PullAddress)
	if err != nil {return nil, err}
	err = gitRepo.Fetch("origin")
	if err != nil {return nil, err}
	if repo.gpg {
		if ok, errOrigin := gitRepo.VerifyCommit("remotes/origin/master"); ok {
			err = gitRepo.Checkout("remotes/origin/master")
			if err != nil {return nil, err}
		} else {// fallback to last local version
			if ok, err := gitRepo.VerifyCommit("master"); ok {
				err = gitRepo.Checkout("heads/master")
				if err != nil {return nil, err}
			} else {
				return nil, fmt.Errorf("couldn't gpg-verify remote nor local git head: %s|%s", err, errOrigin)
			}
		}
	} else {
		err = gitRepo.Checkout("remotes/origin/master")
		if err != nil {return nil, err}
	}
	err = gitRepo.SubmoduleInit()
	if err != nil {return nil, err}
	err = gitRepo.SubmoduleSync()
	if err != nil {return nil, err}
	err = gitRepo.SubmoduleUpdate()
	if err != nil {return nil, err}
	repo.repo = &gitRepo
	return &repo,err
}

func (r *Repo)Update() error{
	err := r.repo.Fetch("origin")
	if err != nil {return err}
	if r.gpg {
		if ok, err := r.repo.VerifyCommit("remotes/origin/master"); !ok {
			return fmt.Errorf("error verifying commit: %s", err)
		}
	}
	err = r.repo.Checkout("remotes/origin/master")
	if err != nil {return err}
	err = r.repo.Clean("--force","-x",)
	err = r.repo.SubmoduleSync()
	if err != nil {return err}
	err = r.repo.SubmoduleUpdate()
	return err
}
