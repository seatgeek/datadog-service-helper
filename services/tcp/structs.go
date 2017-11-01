package tcp

// See https://github.com/DataDog/integrations-core/tree/master/redisdb

// Config ...
type Config struct {
	InitConfig []string      `yaml:"init_config,flow"`
	Instances  []*ConfigItem `yaml:"instances"`
}

// ConfigItem ...
type ConfigItem struct {
	Name                string   `yaml:"name"`
	Host                string   `yaml:"host"`
	Port                int      `yaml:"port"`
	Timeout             int      `yaml:"timeout"`
	CollectResponseTime bool     `yaml:"collect_response_time"`
	Tags                []string `yaml:"tags"`
}
