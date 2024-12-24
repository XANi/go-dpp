# DPP

Distributed puppet runner (WiP)

A small toolbox for running distributed puppet (each node downloads repo and runs it without central puppet master)

Currently implemented features:

* Git as VCS
* Git GPG signing supported (list of allowed keys per repo)
* Ability to prepare deploy package ( `dpp package`, output in `/tmp/dpp.tar.gz` )
* Locking (only one puppet run at a time even if multiple instances are running)
* Signal support (USR1 to run puppet, USR2 to run repo update

Todo:

* ability to connect to main demon and run with "hijacked" output (puppet running by demon piping output to CLI tool)
* some kind of distributed KV support (+ Puppet glue code) to support inter-node coordination
* post last run stat to other node(s) for monitoring
* branch support


## Install

required/recommended packages

    apt install aptitude gnupg2 ca-certificates puppet git puppet-module-puppetlabs-cron-core puppet-module-puppetlabs-host-core puppet-module-puppetlabs-mount-core puppet-module-puppetlabs-sshkeys-core


## Building

    make

Then put [example config](config.example.yaml) in `/etc/dpp/config.yaml` and run it

## How it works

* Git fetch repos
* If gpg is enabled, verify `origin/` branch for commit (requires last commit on top of branch to be signed)
* if commit is ok check it out to local dir
* generate a list of module paths for puppet, then run puppet with generated module and manifest path, repeat
