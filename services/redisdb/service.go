package redisdb

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

// Observe changes in Consul catalog for redisdb
func Observe(payload *cfg.ServicePayload) {
	filePath := os.Getenv("REDISDB_CONFIG_FILE")
	if filePath == "" {
		filePath = "/etc/dd-agent/conf.d/redisdb.yaml"
	}

	currentHash, err := cfg.HashFileMd5(filePath)
	if err != nil {
		logger.Warnf("[redisdb] Could not get initial hash for %s: %s", filePath, err)
		currentHash = ""
	}

	logger.Infof("[redisdb] Existing file hash %s: %s", filePath, currentHash)

	stream := payload.Services.Observe()

	for {
		select {
		case <-payload.QuitCh:
			logger.Warn("[redisdb] stopping")
			return

		case <-stream.Changes():
			stream.Next()

			t := &Config{}

			services := stream.Value().(map[string]*consul.AgentService)

			for _, service := range services {
				if !cfg.ServiceEnabled("redisdb", service.Tags) {
					logger.Debugf("[redisdb] Service %s does not contain 'dd-redisdb' tag", service.Service)
					continue
				}
				logger.Infof("[redisdb] Service %s tags does contain 'dd-redisdb'", service.Service)

				check := &ConfigItem{
					Host: service.Address,
					Port: service.Port,
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
				logger.Fatalf("[redisdb] could not marshal yaml: %v", err)
			}

			reloadRequired, newHash := cfg.WriteIfChange("redisdb", filePath, data, currentHash)
			if !reloadRequired {
				currentHash = newHash
				continue
			}

			payload.ReloadCh <- cfg.ReloadPayload{
				Service: "redisdb",
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
func (a serviceSorter) Less(i, j int) bool { return a[i].Host < a[j].Host && a[i].Port < a[j].Port }
