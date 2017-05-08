package phpfpm

import (
	"fmt"
	"os"
	"sort"

	consul "github.com/hashicorp/consul/api"
	cfg "github.com/seatgeek/datadog-service-helper/config"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var (
	logger = logrus.New()
)

// continuously monitor the local agent services for php-fpm services
// and register them to the local datadog client
func Observe(payload *cfg.ServicePayload) {
	filePath := os.Getenv("PHP_FPM_CONFIG_FILE")
	if filePath == "" {
		filePath = "/etc/dd-agent/conf.d/php_fpm.yaml"
	}

	currentHash, err := cfg.HashFileMd5(filePath)
	if err != nil {
		logger.Warnf("[php-fpm] Could not get initial hash for %s: %s", filePath, err)
		currentHash = ""
	}

	logger.Infof("[php-fpm] Existing file hash %s: %s", filePath, currentHash)

	stream := payload.Services.Observe()

	for {
		select {
		case <-payload.QuitCh:
			logger.Warn("[php-fpm] Stopping")
			return

		case <-stream.Changes():
			stream.Next()

			t := &Config{}

			services := stream.Value().(map[string]*consul.AgentService)

			for _, service := range services {
				if !cfg.ServiceEnabled("php-fpm", service.Tags) {
					logger.Debugf("[php-fpm] Service %s does not contain 'dd-php-fpm' tag", service.Service)
					continue
				}
				logger.Infof("[php-fpm] Service %s tags does contain 'dd-php-fpm'", service.Service)

				projectName := service.Service

				check := &ConfigITem{}
				check.PingURL = fmt.Sprintf("http://%s:%d/php-fpm/%s/%s/%d/ping", service.Address, payload.ListenPort, projectName, service.Address, service.Port)
				check.PingReply = "pong"
				check.StatusURL = fmt.Sprintf("http://%s:%d/php-fpm/%s/%s/%d/status", service.Address, payload.ListenPort, projectName, service.Address, service.Port)
				check.Tags = []string{
					fmt.Sprintf("project:%s", projectName),
				}

				t.Instances = append(t.Instances, check)

				logger.Infof("[php-fpm] Service %s does match '-php-fpm' suffix", service.Service)
			}

			// Sort the services by name so we get consistent output across runs
			sort.Sort(ServiceSorter(t.Instances))

			data, err := yaml.Marshal(&t)
			if err != nil {
				logger.Fatalf("[php-fpm] Could not marshal yaml: %v", err)
				break
			}

			reloadRequired, newHash := cfg.WriteIfChange("php-fpm", filePath, data, currentHash)
			if !reloadRequired {
				continue
			}

			payload.ReloadCh <- cfg.ReloadPayload{
				Service: "php-fpm",
				OldHash: currentHash,
				NewHash: newHash,
			}

			currentHash = newHash
		}
	}
}
