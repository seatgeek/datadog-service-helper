package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"os/signal"
	"syscall"
	"time"

	cfg "github.com/seatgeek/datadog-service-helper/config"

	reloader "github.com/seatgeek/datadog-service-helper/reloader"
	go_exprvar "github.com/seatgeek/datadog-service-helper/services/goexprvar"
	php_fpm "github.com/seatgeek/datadog-service-helper/services/phpfpm"
	redisdb "github.com/seatgeek/datadog-service-helper/services/redisdb"

	"github.com/gorilla/mux"
	consul "github.com/hashicorp/consul/api"
	observer "github.com/imkira/go-observer"
	"github.com/sirupsen/logrus"
	graceful "gopkg.in/tylerb/graceful.v1"
	yaml "gopkg.in/yaml.v2"
)

var logger = logrus.New()
var consulServices = observer.NewProperty(make(map[string]*consul.AgentService, 0))
var listenPort = getListenPort()

func main() {
	logger.Info("Starting datadog monitoring ")

	// Create consul client
	config := consul.DefaultConfig()
	client, err := consul.NewClient(config)
	if err != nil {
		logger.Fatalf("Could not connect to Consul backend: %s", err)
	}

	// Get local agent information
	self, err := client.Agent().Self()
	if err != nil {
		logger.Fatalf("Could not look up self(): %s", err)
	}

	// look up the agent node name
	nodeName := self["Config"]["NodeName"].(string)
	logger.Infof("Connected to Consul node: %s", nodeName)

	// create quitCh
	quitCh := make(cfg.QuitChannel)
	reloadCh := make(cfg.ReloadChannel, 10)

	// create service payload sent to all backends
	payload := &cfg.ServicePayload{
		NodeName:   nodeName,
		Services:   consulServices,
		ListenPort: listenPort,
		QuitCh:     quitCh,
		ReloadCh:   reloadCh,
	}

	reloader := reloader.NewReloader(payload)

	// start monitoring of consul services
	go monitor(client, quitCh)

	// start the reloader
	go reloader.Start()

	// start service observers
	go php_fpm.Observe(payload)
	go go_exprvar.Observe(payload)
	go redisdb.Observe(payload)

	// start the http reserver that proxies http requests to php-cgi
	router := mux.NewRouter()
	router.Handle("/debug/vars", http.DefaultServeMux)
	router.HandleFunc("/datadog/expvar", showExprVar)
	router.HandleFunc("/php-fpm/{project}/{ip}/{port}/{type}", php_fpm.Proxy)

	logger.Infof("")
	logger.Info("Entrypoints:")
	logger.Infof("  - http://127.0.0.1:%d/debug/vars", listenPort)
	logger.Infof("  - http://127.0.0.1:%d/datadog/expvar", listenPort)
	logger.Infof("  - http://127.0.0.1:%d/php-fpm/{project}/{ip}/{port}/{type}", listenPort)
	logger.Infof("")

	// create logger for http server
	w := logger.Writer()
	defer w.Close()

	server := &graceful.Server{
		Server: &http.Server{
			Addr:    fmt.Sprintf(":%d", listenPort),
			Handler: router,
		},
		Timeout:          5 * time.Second,
		TCPKeepAlive:     5 * time.Second,
		Logger:           log.New(w, "HTTP: ", 0),
		NoSignalHandling: true,
	}

	// setup signal handlers
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		signal := <-signalCh
		logger.Warnf("We got signal: %s", signal.String())

		logger.Warn("Closing quitCh")
		close(quitCh)

		logger.Warn("Telling HTTP server to shut down")
		server.Stop(5 * time.Second)

		logger.Info("Shutdown complete")
	}()

	// start the HTTP server (this will block)
	err = server.ListenAndServe()
	if err != nil {
		logger.Fatal(err)
		return
	}

	logger.Info("end of program")
}

// Monitor consul services and emit updates when they change
func monitor(client *consul.Client, quitCh chan string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-quitCh:
			logger.Warn("Stopping monitorConsulServices")
			return

		case <-ticker.C:
			services, err := client.Agent().Services()
			if err != nil {
				logger.Warnf("Could not fetch Consul services: %s", err)
			}

			consulServices.Update(services)
		}
	}
}

func getListenPort() int {
	port := os.Getenv("NOMAD_PORT_http")
	if port == "" {
		return 4000
	}

	i, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		logger.Fatalf(err.Error())
		return -100
	}

	return int(i)
}

func showExprVar(w http.ResponseWriter, r *http.Request) {
	metrics := make([]map[string]string, 0)
	metrics = append(metrics, map[string]string{"path": "php_fpm_instances"})
	metrics = append(metrics, map[string]string{"path": "datadog_reload"})

	config := struct {
		ExpvarURL string              `yaml:"expvar_url"`
		Tags      []string            `yaml:"tags"`
		Metrics   []map[string]string `yaml:"metrics"`
	}{
		"http://127.0.0.1:" + string(listenPort) + "/debug/vars",
		[]string{"project:datadog-monitor"},
		metrics,
	}

	resp, err := yaml.Marshal(&config)
	if err != nil {
		message := fmt.Sprintf("[showExprVar] Could not marshal YAML: %s", err)
		logger.Errorf(message)
		http.Error(w, message, 500)
		return
	}

	text := string(resp)
	text = "---\n" + text

	resp = []byte(text)

	w.Header().Add("Content-Type", "text/yaml")
	w.Write(resp)
}
