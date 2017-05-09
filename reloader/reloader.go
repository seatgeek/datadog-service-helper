package reloader

import (
	"expvar"
	"os"
	"os/exec"
	"sync"
	"time"

	cfg "github.com/seatgeek/datadog-service-helper/config"
	"github.com/sirupsen/logrus"
)

type Reloader struct {
	payload      *cfg.ServicePayload
	shouldReload bool
	mutex        sync.Mutex
}

var logger = logrus.New()
var reloadCounter = expvar.NewInt("datadog_agent_reloads")

func NewReloader(payload *cfg.ServicePayload) *Reloader {
	return &Reloader{
		payload:      payload,
		shouldReload: false,
	}
}

func (r *Reloader) Start() {
	timer := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-r.payload.QuitCh:
			return

		case <-r.payload.ReloadCh:
			r.mutex.Lock()

			logger.Info("Marking datadog-agent for reloading")
			r.shouldReload = true

			r.mutex.Unlock()

		case <-timer.C:
			r.mutex.Lock()

			logger.Debugf("Reloader ticker start")

			if r.shouldReload {
				r.reloadDataDogService()
				r.shouldReload = false
			}

			logger.Debugf("Reloader ticker stop")

			r.mutex.Unlock()
		}
	}
}

func (r *Reloader) reloadDataDogService() {
	reloadCounter.Add(1)

	if os.Getenv("DONT_RELOAD_DATADOG") != "" {
		logger.Infof("Not reloading datadog-agent (env: DONT_RELOAD_DATADOG)")
		return
	}

	logger.Warn("Reloading datadog-agent")

	cmd := "/usr/sbin/service"
	args := []string{"datadog-agent", "reload"}
	if err := exec.Command(cmd, args...).Run(); err != nil {
		logger.Fatalf("Failed to reload datadog-agent: %s", err)
	}

	logger.Infof("Successfully reloaded datadog-agent")
}
