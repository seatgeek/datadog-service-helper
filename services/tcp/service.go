package tcp

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

// Observe changes in Consul catalog for TCPcheck
func Observe(payload *cfg.ServicePayload) {
	filePath := os.Getenv("TCP_CHECK_CONFIG_FILE")
	if filePath == "" {
		filePath = "/etc/dd-agent/conf.d/tcp_check.yaml"
	}

	currentHash, err := cfg.HashFileMd5(filePath)
	if err != nil {
		logger.Warnf("[TCPcheck] Could not get initial hash for %s: %s", filePath, err)
		currentHash = ""
	}

	logger.Infof("[TCPcheck] Existing file hash %s: %s", filePath, currentHash)

	stream := payload.Services.Observe()

	for {
		select {
		case <-payload.QuitCh:
			logger.Warn("[TCPcheck] stopping")
			return

		case <-stream.Changes():
			stream.Next()

			t := &Config{}

			services := stream.Value().(map[string]*consul.AgentService)

			for _, service := range services {
				if !cfg.ServiceEnabled("tcp-check", service.Tags) {
					logger.Debugf("[TCPcheck] Service %s does not contain 'dd-tcp-check' tag", service.Service)
					continue
				}
				logger.Infof("[TCPcheck] Service %s tags does contain 'dd-tcp-check'", service.Service)

				check := &ConfigItem{
					Name:                service.Service,
					Host:                service.Address,
					Port:                service.Port,
					Timeout:             5,
					CollectResponseTime: true,
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
				logger.Fatalf("[TCPcheck] could not marshal yaml: %v", err)
			}

			reloadRequired, newHash := cfg.WriteIfChange("TCPcheck", filePath, data, currentHash)
			if !reloadRequired {
				currentHash = newHash
				continue
			}

			payload.ReloadCh <- cfg.ReloadPayload{
				Service: "TCPcheck",
				OldHash: currentHash,
				NewHash: newHash,
			}

			currentHash = newHash
		}
	}
}

// serviceSorter sorts planets by Host + Port
type serviceSorter []*ConfigItem

func (a serviceSorter) Len() int           { return len(a) }
func (a serviceSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a serviceSorter) Less(i, j int) bool { return a[i].Port < a[j].Port }
