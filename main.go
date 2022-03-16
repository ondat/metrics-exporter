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
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	address  = ":9100"
	endpoint = "/metrics"
)

// globals
var apiSecretsPath string

func main() {
	var (
		timeoutFlag        = flag.Int("timeout", 5, "Timeout in seconds to serve metrics.")
		apiSecretsPathFlag = flag.String("api-secrets-path", "/etc/storageos/secrets/api", "Path where the StorageOS api secrets are mounted. The secret must have \"username\" and \"password\" set.")
	)

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	apiSecretsPath = *apiSecretsPathFlag

	log := zap.New(zap.UseFlagOptions(&opts))

	prometheusRegistry := prometheus.NewRegistry()
	prometheusRegistry.Register(NewDiskStatsCollector(log))
	prometheusRegistry.Register(NewFileSystemCollector(log))

	// k8s endpoints
	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/readyz", readyz)

	// metrics page
	http.Handle(endpoint, promhttp.HandlerFor(
		prometheusRegistry,
		promhttp.HandlerOpts{
			// the request continues on the background but the user gets the correct response
			Timeout:       time.Second * time.Duration(*timeoutFlag),
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
		templ.Execute(w, &data)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	}))

	log.Info("starting http handler", "port", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Error(err, "problem running http server")
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
