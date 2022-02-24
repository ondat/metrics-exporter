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
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var (
		address = flag.String("listen-address", ":9100", "The address to listen on for HTTP requests.")
	)

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))

	// metrics registry, relying on Prometheus implementation to keep it simple
	prometheusRegistry := prometheus.NewRegistry()
	prometheusRegistry.Register(NewDiskStatsCollector())

	// http endpoint handler
	http.Handle("/metrics", promhttp.HandlerFor(prometheusRegistry, promhttp.HandlerOpts{})) // test/fix logger param

	logger.Info("starting http handler", "port", address)
	if err := http.ListenAndServe(*address, nil); err != nil {
		logger.Error(err, "problem running http server")
		os.Exit(1)
	}
}
