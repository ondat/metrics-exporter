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
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	configondatv1 "github.com/ondat/metrics-exporter/api/config.storageos.com/v1"
)

func readConfigFile(path string) (*configondatv1.MetricsExporterConfig, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read file at %s: %v", path, err)
	}

	codecs := serializer.NewCodecFactory(scheme)

	cfg := (&configondatv1.MetricsExporterConfig{}).Default()
	if err = runtime.DecodeInto(codecs.UniversalDecoder(), content, cfg); err != nil {
		return nil, fmt.Errorf("could not decode file into runtime.Object: %v", err)
	}

	return cfg, nil
}

func getConfigOrDie() (path string, cfg configondatv1.MetricsExporterConfig) {
	var configFile string
	var logLevelFlag string
	var timeoutFlag int

	defaults := (&configondatv1.MetricsExporterConfig{}).Default()

	flag.StringVar(&configFile, "config", "",
		"The exporter will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	flag.StringVar(&logLevelFlag, "log-level", defaults.LogLevel,
		"Verbosity of log messages. Accepts go.uber.org/zap log levels.")
	flag.IntVar(&timeoutFlag, "timeout", defaults.Timeout, "Timeout in seconds to serve metrics.")
	flag.Parse()

	if len(configFile) > 0 {
		parsedCfg, err := readConfigFile(configFile)
		if err != nil {
			log.Printf("failed to load config from file \"%s\": %s\n", configFile, err.Error())
			os.Exit(1)
		}
		cfg = *parsedCfg
	} else {
		cfg = *defaults
	}

	// override defaults/configmap with the supplied flag values
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "log-level":
			cfg.LogLevel = logLevelFlag
		case "timeout":
			cfg.Timeout = timeoutFlag
		}
	})

	return configFile, cfg
}
