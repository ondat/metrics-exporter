package main

import "github.com/prometheus/client_golang/prometheus"

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
	// SCRAPE_SUBSYSTEM defines the category about the metrics gathering process
	// itself (success, failures, duration, etc)
	//
	// "ondat_scrape_..."
	SCRAPE_SUBSYSTEM = "scrape"
)

var (
	// label present in all PVC metrics to indicate which PVC its values refer to
	labelNames = []string{"pvc"}

	// scrapeDurationDesc defines the scrape duration metric desc
	// shared between all metric collectors
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(ONDAT_NAMESPACE, SCRAPE_SUBSYSTEM, "collector_duration_seconds"),
		"Duration of a collector scrape.",
		[]string{"collector"}, nil,
	)

	// scrapeDurationDesc defines the scrape success/failure metric desc
	// shared between all metric collectors
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(ONDAT_NAMESPACE, SCRAPE_SUBSYSTEM, "collector_success"),
		"Whether a collector succeeded.",
		[]string{"collector"}, nil,
	)
)

// Metric is a wrapper over prometheus types (desc and type) defining a
// standalone metric
type Metric struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}
