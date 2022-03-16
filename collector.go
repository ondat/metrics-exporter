package main

import "github.com/prometheus/client_golang/prometheus"

type Collector struct {
}

func NewCollector() Collector {
	return Collector{}
}

func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationMetric.desc
	ch <- scrapeSuccessMetric.desc
}

func (c Collector) Collect(ch chan<- prometheus.Metric) {

}
