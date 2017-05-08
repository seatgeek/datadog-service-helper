package redisdb

// See https://github.com/DataDog/integrations-core/tree/master/redisdb

// Config ...
type Config struct {
	InitConfig []string      `yaml:"init_config,flow"`
	Instances  []*ConfigItem `yaml:"instances"`
}

// ConfigItem ...
type ConfigItem struct {
	Host string   `yaml:"host"`
	Port int      `yaml:"port"`
	Tags []string `yaml:"tags"`
}
