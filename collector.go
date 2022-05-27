package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Collector interface {
	Name() string
	Collect(log *zap.SugaredLogger, ch chan<- prometheus.Metric, ondatVolumes []*Volume) error
}

type CollectorGroup struct {
	log *zap.SugaredLogger

	collectors []Collector
}

func NewCollectorGroup(log *zap.SugaredLogger, c []Collector) CollectorGroup {
	return CollectorGroup{
		log:        log,
		collectors: c,
	}
}

func (c CollectorGroup) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationMetric.desc
	ch <- scrapeSuccessMetric.desc
}

// Collect gathers all the metrics and reports back on both the process itself
// but also everything that has been gathered successfully.
// Can be called multiple times asynchronously from the prometheus default registry.
func (c CollectorGroup) Collect(ch chan<- prometheus.Metric) {
	// All local Ondat volumes fetched from the state files
	ondatVolumes, err := GetVolumesFromLocalState(c.log)
	if err != nil {
		c.log.Errorw("failed to get Ondat volumes from local state files", "error", err)
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(c.collectors))
	for _, collector := range c.collectors {
		// each collector gathers metrics is parallel
		go func(collector Collector) {
			log := c.log.With("req_id", uuid.New())
			execute(log, collector, ch, ondatVolumes)
			wg.Done()
		}(collector)
	}
	wg.Wait()
}

func execute(log *zap.SugaredLogger, c Collector, ch chan<- prometheus.Metric, ondatVolumes []*Volume) {
	timeStart := time.Now()

	// best effort
	// even if there's an error processing a specific Volume or disk
	// all those that succeed still get reported
	err := c.Collect(log, ch, ondatVolumes)

	duration := time.Since(timeStart)
	ch <- prometheus.MustNewConstMetric(scrapeDurationMetric.desc, scrapeDurationMetric.valueType, duration.Seconds(), c.Name())

	var success float64
	if err != nil {
		log.Errorw("collector failed", "collector", c.Name())
		success = 0
	} else {
		log.Debugw("collector succeeded", "collector", c.Name())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeSuccessMetric.desc, scrapeSuccessMetric.valueType, success, c.Name())
}
