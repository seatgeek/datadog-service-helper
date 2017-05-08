package phpfpm

// ServiceSorter sorts planets by PingURL
type ServiceSorter []*ConfigITem

func (a ServiceSorter) Len() int           { return len(a) }
func (a ServiceSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ServiceSorter) Less(i, j int) bool { return a[i].PingURL < a[j].PingURL }

// Config ...
type Config struct {
	InitConfig []string      `yaml:"init_config,flow"`
	Instances  []*ConfigITem `yaml:"instances"`
}

// ConfigITem ...
type ConfigITem struct {
	StatusURL string   `yaml:"status_url"`
	PingURL   string   `yaml:"ping_url"`
	PingReply string   `yaml:"ping_reply"`
	Tags      []string `yaml:"tags"`
}
