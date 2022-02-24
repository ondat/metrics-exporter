package main

import "github.com/prometheus/client_golang/prometheus"

const (
	// ONDAT_NAMESPACE defines the common namespace to be used by all our metrics.
	//
	// "ondat_..."
	ONDAT_NAMESPACE = "ondat"
	// DISK_SUBSYSTEM defines the common category shared between all metrics we
	// expose related to PVCs (only ones we support right now)
	//
	// "ondat_disk_..."
	DISK_SUBSYSTEM = "disk"
	// SCRAPE_SUBSYSTEM defines the category about the metrics gathering process
	// itself (success, failures, duration, etc)
	//
	// "ondat_scrape_..."
	SCRAPE_SUBSYSTEM = "scrape"
	// SECONDS_PER_TICK defines ...
	SECONDS_PER_TICK = 1.0 / 1000.0
)

var (
	labelNames = []string{"pvc"}

	// scrapeDurationDesc defines the scrape duration metric desc
	// shared between metrics collectors
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(ONDAT_NAMESPACE, SCRAPE_SUBSYSTEM, "collector_duration_seconds"),
		"Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)

	// scrapeDurationDesc defines the scrape success/failure metric desc
	// shared between metrics collectors
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(ONDAT_NAMESPACE, SCRAPE_SUBSYSTEM, "collector_success"),
		"Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

// Metric is a wrapper over prometheus types (desc and type) defining a standalone metric
type Metric struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}
