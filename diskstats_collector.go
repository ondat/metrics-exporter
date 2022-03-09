package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
)

// SECOND_IN_MILLISECONDS defines the number of seconds on a milliseconds. Used
// to transform metrics that express a duration in milliseconds.
const SECOND_IN_MILLISECONDS = 1.0 / 1000.0

// DiskStatsCollector implements the prometheus Collector interface
// Its sole responsability is gathering metrics on PVCs
type DiskStatsCollector struct {
	log            logr.Logger
	apiSecretsPath string

	// info metrics of all the scraped PVCs
	infoDesc Metric
	// all PVC metrics we gather from diskstats
	// usefull as a standalone variable to iterate over and index match with diskstats's content
	// order MUST match the columns in the diskstats file
	descs []Metric
}

func NewDiskStatsCollector(log logr.Logger, apiSecretsPath string) DiskStatsCollector {
	return DiskStatsCollector{
		log:            log,
		apiSecretsPath: apiSecretsPath,
		infoDesc: Metric{
			desc: prometheus.NewDesc(prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "info"),
				"Info of Ondat volumes and devices.",
				[]string{"device", "pvc", "major", "minor"}, nil,
			),
			valueType: prometheus.GaugeValue,
		},
		descs: []Metric{
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "reads_completed_total"),
					"The total number of reads completed successfully.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "reads_merged_total"),
					"The total number of reads merged.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "read_bytes_total"),
					"The total number of bytes read successfully.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "read_time_seconds_total"),
					"The total number of seconds spent by all reads.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "writes_completed_total"),
					"The total number of writes completed successfully.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "writes_merged_total"),
					"The number of writes merged.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "written_bytes_total"),
					"The total number of bytes written successfully.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "write_time_seconds_total"),
					"This is the total number of seconds spent by all writes.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "io_now"),
					"The number of I/Os currently in progress.",
					labelNames, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "io_time_seconds_total"),
					"Total seconds spent doing I/Os.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "io_time_weighted_seconds_total"),
					"The weighted # of seconds spent doing I/Os.",
					labelNames, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "discards_completed_total"),
					"The total number of discards completed successfully.",
					labelNames, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "discards_merged_total"),
					"The total number of discards merged.",
					labelNames, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "discarded_sectors_total"),
					"The total number of sectors discarded successfully.",
					labelNames, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "discard_time_seconds_total"),
					"This is the total number of seconds spent by all discards.",
					labelNames, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "flush_requests_total"),
					"The total number of flush requests completed successfully",
					labelNames, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "flush_requests_time_seconds_total"),
					"This is the total number of seconds spent by all flush requests.",
					labelNames, nil,
				), valueType: prometheus.CounterValue,
			},
		},
	}
}

func (c DiskStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect gathers all the metrics and reports back on both the process itself
// but also everything that has been gathered successfully.
// Can be called multiple times asynchronously from the registry.
func (c DiskStatsCollector) Collect(ch chan<- prometheus.Metric) {
	timeStart := time.Now()

	if err := ValidateDir(STOS_VOLUMES_PATH); err != nil {
		c.log.Error(err, "error validating Ondat volumes directory")
		ReportScrapeResult(c.log, ch, timeStart, false)
		return
	}

	volumesOnNode, err := GetOndatVolumes()
	if err != nil {
		c.log.Error(err, "error getting Ondat volumes")
		ReportScrapeResult(c.log, ch, timeStart, false)
		return
	}

	if len(volumesOnNode) == 0 {
		c.log.Info("no Ondat volumes")
		// TODO confirm this behaviour is desired
		ReportScrapeResult(c.log, ch, timeStart, true)
		return
	}

	diskstats, err := ProcDiskstats()
	if err != nil {
		c.log.Error(err, "error reading diskstats")
		ReportScrapeResult(c.log, ch, timeStart, false)
		return
	}

	// All Ondat volumes fetched from the storageos container's API
	// Cluster wide thus only one request is needed
	OndatVolumes := []VolumePVC{}

	for _, vol := range volumesOnNode {
		err = GetOndatVolumeState(vol)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				c.log.Error(err, fmt.Sprintf("error reading volume %s state file", vol.ID))
				continue
			}

			// state files are only present on nodes that host either the master of replica
			// deployments of a volume. If the volume is attached on a node where neither
			// of those is found we won't have any state files thus fallback to requesting
			// the data from the storageos API on the same node

			// cluster wide thus only one request needed
			if len(OndatVolumes) == 0 {
				OndatVolumes, err = GetAllOndatVolumes(c.log, c.apiSecretsPath)
				if err != nil {
					continue
				}
			}

			for _, apiVol := range OndatVolumes {
				if vol.ID == apiVol.ID {
					vol.PVC = apiVol.PVC
				}
			}
		}

		for _, stats := range diskstats {
			// match with Ondat volume through diskstat row's Major and Minor numbers
			if vol.Major != int(stats.MajorNumber) || vol.Minor != int(stats.MinorNumber) {
				continue
			}
			vol.metrics = stats

			ch <- NewConstMetric(c.log, c.infoDesc.desc, c.infoDesc.valueType, 1.0, stats.DeviceName, vol.PVC, fmt.Sprint(vol.Major), fmt.Sprint(vol.Minor))

			diskSectorSize := 512.0
			logicalBlockSize, err := GetBlockDeviceLogicalBlockSize(stats.DeviceName)
			if err != nil {
				c.log.Error(err, "error reading device logical block size, falling back to default")
				// continue with default sector size
			} else {
				diskSectorSize = float64(logicalBlockSize)
			}

			// total diskstats record count, less the MajorNumber, MinorNumber and DeviceName
			statCount := stats.IoStatsCount - 3

			for i, val := range []float64{
				float64(stats.ReadIOs),
				float64(stats.ReadMerges),
				float64(stats.ReadSectors) * diskSectorSize,
				float64(stats.ReadTicks) * SECOND_IN_MILLISECONDS,
				float64(stats.WriteIOs),
				float64(stats.WriteMerges),
				float64(stats.WriteSectors) * diskSectorSize,
				float64(stats.WriteTicks) * SECOND_IN_MILLISECONDS,
				float64(stats.IOsInProgress),
				float64(stats.IOsTotalTicks) * SECOND_IN_MILLISECONDS,
				float64(stats.WeightedIOTicks) * SECOND_IN_MILLISECONDS,
				float64(stats.DiscardIOs),
				float64(stats.DiscardMerges),
				float64(stats.DiscardSectors),
				float64(stats.DiscardTicks) * SECOND_IN_MILLISECONDS,
				float64(stats.FlushRequestsCompleted),
				float64(stats.TimeSpentFlushing) * SECOND_IN_MILLISECONDS,
			} {
				if i >= statCount {
					// didn't read all the above fields from diskstats
					// kernel version lower than 5.5
					break
				}

				ch <- NewConstMetric(c.log, c.descs[i].desc, c.descs[i].valueType, val, vol.PVC)
			}
		}
	}
	ReportScrapeResult(c.log, ch, timeStart, true)
}

func ReportScrapeResult(log logr.Logger, ch chan<- prometheus.Metric, timer time.Time, success bool) {
	ch <- NewConstMetric(log, scrapeDurationDesc, prometheus.GaugeValue, time.Since(timer).Seconds(), "diskstats")

	successReturn := 1.0
	if !success {
		successReturn = 0
	}
	ch <- NewConstMetric(log, scrapeSuccessDesc, prometheus.GaugeValue, successReturn, "diskstats")
}

func NewConstMetric(log logr.Logger, desc *prometheus.Desc, valueType prometheus.ValueType, value float64, labels ...string) prometheus.Metric {
	m, err := prometheus.NewConstMetric(desc, valueType, value, labels...)
	if err != nil {
		log.Error(err, "failed creating new const metric: %w", err)
	}
	return m
}
