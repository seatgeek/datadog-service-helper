package rabbitmq

import (
	"fmt"
	"os"
	"sort"

	consul "github.com/hashicorp/consul/api"
	cfg "github.com/seatgeek/datadog-service-helper/config"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var logger = logrus.New()

// Observe changes in Consul catalog for rabbitmq
func Observe(payload *cfg.ServicePayload) {
	filePath := os.Getenv("RABBITMQ_CONFIG_FILE")
	if filePath == "" {
		filePath = "/etc/dd-agent/conf.d/rabbitmq.yaml"
	}
	password := os.Getenv("RABBITMQ_PASSWORD")
	if password == "" {
		password = "guest"
	}
	username := os.Getenv("RABBITMQ_USERNAME")
	if username == "" {
		username = "guest"
	}

	currentHash, err := cfg.HashFileMd5(filePath)
	if err != nil {
		logger.Warnf("[rabbitmq] Could not get initial hash for %s: %s", filePath, err)
		currentHash = ""
	}

	logger.Infof("[rabbitmq] Existing file hash %s: %s", filePath, currentHash)

	stream := payload.Services.Observe()

	for {
		select {
		case <-payload.QuitCh:
			logger.Warn("[rabbitmq] stopping")
			return

		case <-stream.Changes():
			stream.Next()

			t := &Config{}

			services := stream.Value().(map[string]*consul.AgentService)

			for _, service := range services {
				if !cfg.ServiceEnabled("rabbitmq", service.Tags) {
					logger.Debugf("[rabbitmq] Service %s does not contain 'dd-rabbitmq' tag", service.Service)
					continue
				}
				logger.Infof("[rabbitmq] Service %s tags does contain 'dd-rabbitmq'", service.Service)

				check := &ConfigItem{
					URL:      fmt.Sprintf("http://%s:%s/api/", service.Address, service.Port),
					Username: username,
					Password: password,
					Tags: []string{
						fmt.Sprintf("service:%s", service.Service),
					},
				}

				t.Instances = append(t.Instances, check)
			}

			// Sort the services by name so we get consistent output across runs
			sort.Sort(serviceSorter(t.Instances))

			data, err := yaml.Marshal(&t)
			if err != nil {
				logger.Fatalf("[rabbitmq] could not marshal yaml: %v", err)
			}

			reloadRequired, newHash := cfg.WriteIfChange("rabbitmq", filePath, data, currentHash)
			if !reloadRequired {
				currentHash = newHash
				continue
			}

			payload.ReloadCh <- cfg.ReloadPayload{
				Service: "rabbitmq",
				OldHash: currentHash,
				NewHash: newHash,
			}

			currentHash = newHash
		}
	}
}

// serviceSorter sorts planets by ExpvarURL
type serviceSorter []*ConfigItem

func (a serviceSorter) Len() int           { return len(a) }
func (a serviceSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a serviceSorter) Less(i, j int) bool { return a[i].URL < a[j].URL }
