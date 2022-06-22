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
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true

// MetricsExporterConfig is the Schema for the metricsexporterconfigs API
type MetricsExporterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	MetricsExporterConfigSpec `json:",inline"`
}

// MetricsExporterConfigSpec represents the configuration options for the metrics-exporter. These fields shall
// be inlined in the StorageOSCluster.Spec.Metrics.
type MetricsExporterConfigSpec struct {
	// Verbosity of log messages. Accepts go.uber.org/zap log levels.
	// +kubebuilder:default:info
	// +kubebuilder:validation:Enum=debug;info;warn;error;dpanic;panic;fatal
	LogLevel string `json:"logLevel,omitempty"`

	// Timeout in seconds to serve metrics.
	// +kubebuilder:default:10
	// +kubebuilder:validation:Minimum=1
	Timeout int `json:"timeout,omitempty"`

	// DisabledCollectors is a list of collectors that shall be disabled. By default, all are enabled.
	DisabledCollectors []MetricsExporterCollector `json:"disabledCollectors,omitempty"`
}

// MetricsExporterCollector is the name of a metrics collector in the metrics-exporter.
// +kubebuilder:validation:Enum=diskstats;filesystem
type MetricsExporterCollector string

// All known metrics-exporter collectors are listed here.
const (
	MetricsExporterCollectorDiskStats  MetricsExporterCollector = "diskstats"
	MetricsExporterCollectorFileSystem MetricsExporterCollector = "filesystem"
)

func init() {
	SchemeBuilder.Register(&MetricsExporterConfig{})
}
