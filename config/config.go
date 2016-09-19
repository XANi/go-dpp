package config

type Config struct {
	Repo         map[string]Repository `yaml:"repo"`
	UseRepos     []string              `yaml:"use_repos"`
	ManifestFrom string                `yaml:"shared"`
	RepoDir      string                `yaml:"repo_dir"`
	ListenAddr   string                `yaml:"listen_addr"`
	Log          struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
}

type Repository struct{
	Branch string `yaml:"branch"`
    CheckUrl string `yaml:"check_url"`
    Force bool `yaml:"force"`
    GpgKeys []string `yaml:"gpg"`
    PullUrl string `yaml:"pull_url"`
}
