package goexprvar

// Config ...
type Config struct {
	InitConfig []string      `yaml:"init_config,flow"`
	Instances  []*ConfigItem `yaml:"instances"`
}

// ConfigItem ...
type ConfigItem struct {
	ExpvarURL string          `yaml:"expvar_url"`
	Tags      []string        `yaml:"tags"`
	Metrics   []*MetricConfig `yaml:"metrics"`
}

// MetricConfig ...
type MetricConfig map[string]string
