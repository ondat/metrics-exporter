/*
Copyright 2022 Ondat.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	address  = ":9100"
	endpoint = "/metrics"
)

func main() {
	var logLevelFlag, apiSecretsPathFlag string
	var timeoutFlag int

	flag.StringVar(&logLevelFlag, "log-level", "info", "Verbosity of log messages. Accepts go.uber.org/zap log levels.")
	flag.StringVar(&apiSecretsPathFlag, "api-secrets-path", "/etc/storageos/secrets/api", "Path where the StorageOS api secrets are mounted. The secret must have \"username\" and \"password\" set.")
	flag.IntVar(&timeoutFlag, "timeout", 5, "Timeout in seconds to serve metrics.")

	flag.Parse()

	level, err := zapcore.ParseLevel(logLevelFlag)
	if err != nil {
		log.Printf("failed to parse log level %s: %s\n", logLevelFlag, err.Error())
		os.Exit(1)
	}

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	loggerConfig.Level.SetLevel(level)

	logger, err := loggerConfig.Build()
	if err != nil {
		log.Printf("failed to build logger from desired config: %s\n", err.Error())
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()
	log := logger.Sugar()

	metricsCollectors := []Collector{
		NewDiskStatsCollector(),
		NewFileSystemCollector(),
	}

	prometheusRegistry := prometheus.NewRegistry()
	_ = prometheusRegistry.Register(NewCollectorGroup(log, apiSecretsPathFlag, metricsCollectors))

	// k8s endpoints
	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/readyz", readyz)

	// metrics page
	http.Handle(endpoint, promhttp.HandlerFor(
		prometheusRegistry,
		promhttp.HandlerOpts{
			// the request will continue on the background but the user requests
			// gets the correct timeout response
			Timeout:       time.Second * time.Duration(timeoutFlag),
			ErrorHandling: promhttp.ContinueOnError,
		},
	))

	// landing page
	// prometheus.io/docs/instrumenting/writing_exporters/#landing-page
	var templ = template.Must(template.ParseFiles("index.html"))
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title           string
			MetricsEndpoint string
		}{
			Title:           "Metrics exporter",
			MetricsEndpoint: endpoint,
		}
		_ = templ.Execute(w, &data)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	}))

	log.Infow("starting http handler", "port", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Errorw("error running http server", "error", err)
		os.Exit(1)
	}
}

// healthz is a liveness probe.
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// readyz is a readyness probe.
func readyz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
