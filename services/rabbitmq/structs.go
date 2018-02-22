package rabbitmq

// See https://github.com/DataDog/integrations-core/tree/master/rabbitmq

// Config ...
type Config struct {
	InitConfig []string      `yaml:"init_config,flow"`
	Instances  []*ConfigItem `yaml:"instances"`
}

// ConfigItem ...
type ConfigItem struct {
	URL      string   `yaml:"rabbitmq_api_url"`
	Username string   `yaml:"rabbitmq_user"`
	Password string   `yaml:"rabbitmq_pass"`
	Tags     []string `yaml:"tags"`
}
