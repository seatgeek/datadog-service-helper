package goexpvar

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"time"

	consul "github.com/hashicorp/consul/api"
	cache "github.com/patrickmn/go-cache"
	cfg "github.com/seatgeek/datadog-service-helper/config"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var configCache = cache.New(30*time.Minute, 30*time.Second)
var logger = logrus.New()

// Observe changes in Consul catalog for go-expvar
func Observe(payload *cfg.ServicePayload) {
	filePath := os.Getenv("GO_EXPVAR_CONFIG_FILE")
	if filePath == "" {
		filePath = "/etc/dd-agent/conf.d/go_expvar.yaml"
	}

	currentHash, err := cfg.HashFileMd5(filePath)
	if err != nil {
		logger.Warnf("[go-expvar] Could not get initial hash for %s: %s", filePath, err)
		currentHash = ""
	}

	logger.Infof("[go-expvar] Existing file hash %s: %s", filePath, currentHash)

	stream := payload.Services.Observe()

	for {
		select {
		case <-payload.QuitCh:
			logger.Warn("[go-expvar] stopping")
			return

		case <-stream.Changes():
			stream.Next()

			t := &Config{}

			services := stream.Value().(map[string]*consul.AgentService)

			for _, service := range services {
				if !cfg.ServiceEnabled("go-expvar", service.Tags) {
					logger.Debugf("[go-expvar] Service %s does not contain 'dd-go-expvar' tag", service.Service)
					continue
				}
				logger.Infof("[go-expvar] Service %s tags does contain 'dd-go-expvar'", service.Service)

				url := fmt.Sprintf("http://%s:%d/datadog/expvar", service.Address, service.Port)

				check, err := getRemoteConfig(url)
				if err != nil {
					logger.Warnf("[go-expvar] Could not get remote config for %s: %s", url, err)
					continue
				}

				if check.ExpvarURL != "" {
					t.Instances = append(t.Instances, check)
				}
			}

			// Sort the services by name so we get consistent output across runs
			sort.Sort(ServiceSorter(t.Instances))

			data, err := yaml.Marshal(&t)
			if err != nil {
				logger.Fatalf("[go-expvar] could not marshal yaml: %v", err)
			}

			reloadRequired, newHash := cfg.WriteIfChange("go-expvar", filePath, data, currentHash)
			if !reloadRequired {
				currentHash = newHash
				continue
			}

			payload.ReloadCh <- cfg.ReloadPayload{
				Service: "go-expvar",
				OldHash: currentHash,
				NewHash: newHash,
			}

			currentHash = newHash
		}
	}
}

func getRemoteConfig(url string) (config *ConfigItem, err error) {
	cached, found := configCache.Get(url)
	if found {
		config = cached.(*ConfigItem)
		return config, nil
	}

	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Could not GET url '%s': %s", url, err.Error())
	}

	defer response.Body.Close()
	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Could not read response '%s': %s", url, err.Error())
	}

	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("Could not marshal response into YAML '%s', %s", url, err.Error())
	}

	configCache.Set(url, config, cache.DefaultExpiration)
	return config, nil
}

// ServiceSorter sorts planets by ExpvarURL
type ServiceSorter []*ConfigItem

func (a ServiceSorter) Len() int           { return len(a) }
func (a ServiceSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ServiceSorter) Less(i, j int) bool { return a[i].ExpvarURL < a[j].ExpvarURL }
