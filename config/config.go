package config

type Config struct {
	Repo         map[string]Repository `yaml:"repo"`
	UseRepos     []string              `yaml:"use_repos"`
	ManifestFrom string                `yaml:"manifest_from"`
	RepoDir      string                `yaml:"repo_dir"`
	ListenAddr   string                `yaml:"listen_addr"`
	Debug        bool                  `yaml:"debug"`
	RepoPollInterval int `yaml:"poll_interval"`
	Log          struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
	// normally app continues if config errors are reasonably recoverable (so bad push can be fixed remotely
	// that changes it to "die if something is wrong"
	Puppet PuppetInterval `yaml:"puppet"`
	KillOnBadConfig bool `yaml:"kill_on_bad_config"`
}

type PuppetInterval struct {
	StartWait int `yaml:"start_wait"`
	MinimumInterval int `yaml:"minimum_interval"`
	ScheduleRun int `yaml:"schedule_run"`
}

type Repository struct{
	Branch string `yaml:"branch"`
    CheckUrl string `yaml:"check_url"`
    Force bool `yaml:"force"`
    GpgKeys []string `yaml:"gpg"`
    PullUrl string `yaml:"pull_url"`
}
