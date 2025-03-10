package config

type TGSConfig struct {
	Name          string                  `yaml:"name"`
	Subscriptions map[string]Subscription `yaml:"subscriptions"`
}

type Subscription struct {
	RemoteState  RemoteState   `yaml:"remotestate"`
	Environments []Environment `yaml:"environments"`
}

type RemoteState struct {
	Name          string `yaml:"name"`
	ResourceGroup string `yaml:"resource_group"`
}

type Environment struct {
	Name  string `yaml:"name"`
	Stack string `yaml:"stack"`
}

type MainConfig struct {
	Stack Stack `yaml:"stack"`
}

type Stack struct {
	Components   map[string]Component `yaml:"components"`
	Architecture Architecture         `yaml:"architecture"`
}

type Component struct {
	Source   string   `yaml:"source"`
	Deps     []string `yaml:"deps"`
	Provider string   `yaml:"provider"`
	Version  string   `yaml:"version"`
}

type Architecture struct {
	Regions map[string][]RegionComponent `yaml:"regions"`
}

type RegionComponent struct {
	Component string   `yaml:"component"`
	Apps      []string `yaml:"apps"`
}
