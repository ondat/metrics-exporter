package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// ONDAT_NAMESPACE defines the common namespace used by all our metrics.
	//
	// "ondat_..."
	ONDAT_NAMESPACE = "ondat"
	// DISK_SUBSYSTEM defines the common category shared between all metrics we
	// expose about PVCs
	//
	// "ondat_disk_..."
	DISK_SUBSYSTEM = "disk"
	// FILE_SYSTEM_SUBSYSTEM defines the common category shared between all filesystem
	// metrics
	//
	// "ondat_filesystem_..."
	FILE_SYSTEM_SUBSYSTEM = "filesystem"
	// SCRAPE_SUBSYSTEM defines the category about the metrics gathering process
	// itself (success, failures, duration, etc)
	//
	// "ondat_scrape_..."
	SCRAPE_SUBSYSTEM = "scrape"
)

var (
	// labels present in all disk metrics to identify the PVC
	pvcLabels = []string{"pvc"}

	// labels present in all filesystem metrics to identify the device
	fsLabels = []string{"pvc", "device", "fstype", "mountpoint"}

	// labels present in all scrape metrics to identify the collector
	collectorLabels = []string{"collector"}

	// scrapeDurationMetric defines the scrape duration metric
	//
	// shared between all metric collectors
	scrapeDurationMetric = Metric{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(ONDAT_NAMESPACE, SCRAPE_SUBSYSTEM, "collector_duration_seconds"),
			"Duration of a collector scrape.",
			collectorLabels, nil,
		),
		valueType: prometheus.GaugeValue,
	}

	// scrapeDurationDesc defines the scrape success/failure metric
	//
	// shared between all metric collectors
	scrapeSuccessMetric = Metric{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(ONDAT_NAMESPACE, SCRAPE_SUBSYSTEM, "collector_success"),
			"Whether a collector succeeded.",
			collectorLabels, nil,
		),
		valueType: prometheus.GaugeValue,
	}
)

// Metric is a wrapper over prometheus types (desc and type) defining a
// standalone metric
type Metric struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}
