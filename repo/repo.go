package repo

import (
	"fmt"
	"github.com/XANi/go-gitcli"
	"github.com/XANi/go-gpgcli"
	"go.uber.org/zap"
)

type Repo struct {
	branch  string
	address string
	dir     string
	force   bool
	gpg     bool
	gpgKeys []string
	repo    *gitcli.Repo
	l       *zap.SugaredLogger
}

type Config struct {
	PullAddress string
	Branch      string
	TargetDir   string
	GpgKeys     []string
	Force       bool
	Debug       bool
	Logger      *zap.SugaredLogger
}

func New(cfg Config) (r *Repo, err error) {
	var repo Repo
	repo.l = cfg.Logger
	// branch support is not implemented yet: everything is hardcoded to remotes/origin/master
	if cfg.Branch != "" && cfg.Branch != "master" && cfg.Branch != "remotes/origin/master" {
		repo.l.Warnf("branch [%s] requested but branch support is not implemented, using remotes/origin/master", cfg.Branch)
	}
	// resolve fingerprints to their full length
	if len(cfg.GpgKeys) > 0 {
		repo.gpg = true
		repo.gpgKeys = []string{}
		gpg, err := gpgcli.New()
		if err != nil {
			return nil, err
		}
		for _, key := range cfg.GpgKeys {
			fingerprint, err := gpg.GetFingerprintById(key)
			if err != nil {
				repo.l.Errorf("couldn't resolve gpg fingerprint for key %s in local gpg db: %s, ignoring", key, err)
				continue
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
	if err != nil {
		gitRepo.HardReset() //dealing with index smaller than expected
		return nil, err
	}
	err = gitRepo.SetRemote("origin", cfg.PullAddress)
	if err != nil {
		return nil, err
	}
	err = gitRepo.Fetch("origin")
	if err != nil {
		gitRepo.HardReset() //dealing with index smaller than expected
		// we do not err out here coz we want it to work offline too
		repo.l.Errorf("error fetching[%s]: %s", cfg.TargetDir, err)
	}
	if repo.gpg {
		if ok, errOrigin := gitRepo.VerifyCommit("remotes/origin/master"); ok {
			err = gitRepo.Checkout("--force", "remotes/origin/master")
			if err != nil {
				return nil, err
			}
		} else { // fallback to last local version
			repo.l.Errorf("failed gpg-validating remotes/origin/%s:%s", "master", errOrigin)
			if ok, err := gitRepo.VerifyCommit("master"); ok {
				err = gitRepo.Checkout("--force", "heads/master")
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("couldn't gpg-verify remote nor local git head: %s|%s", err, errOrigin)
			}
		}
	} else {
		err = gitRepo.Checkout("--force", "remotes/origin/master")
		if err != nil {
			return nil, err
		}
	}
	err = gitRepo.SubmoduleInit()
	if err != nil {
		return nil, err
	}
	err = gitRepo.SubmoduleSync()
	if err != nil {
		repo.l.Errorf("error syncing submodules[%s]: %s", cfg.TargetDir, err)
	}
	err = gitRepo.SubmoduleUpdate()
	if err != nil {
		repo.l.Errorf("error updating submodules[%s]: %s", cfg.TargetDir, err)
		return nil, err
	}
	repo.repo = &gitRepo
	return &repo, nil
}

func (r *Repo) Update() error {
	err := r.repo.Fetch("origin")
	if err != nil {
		return err
	}
	if r.gpg {
		if ok, err := r.repo.VerifyCommit("remotes/origin/master"); !ok {
			return fmt.Errorf("error verifying commit: %s", err)
		}
	}
	err = r.repo.Checkout("--force", "remotes/origin/master")
	if err != nil {
		return err
	}
	err = r.repo.Clean("--force", "-x")
	if err != nil {
		return err
	}
	err = r.repo.SubmoduleSync()
	if err != nil {
		return err
	}
	err = r.repo.SubmoduleUpdate()
	return err
}
