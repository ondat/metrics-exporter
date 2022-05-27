package main

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	// SECOND_IN_MILLISECONDS defines the number of seconds on a milliseconds. Used
	// to transform metrics that express a duration in milliseconds.
	SECOND_IN_MILLISECONDS   = 1.0 / 1000.0
	DISKSTATS_COLLECTOR_NAME = "diskstats"
)

// DiskStatsCollector implements the prometheus Collector interface
// Its sole responsibility is gathering metrics on PVCs
type DiskStatsCollector struct {
	// info of all the scraped PVCs
	info Metric

	// all PVC metrics we gather from diskstats
	// useful as a standalone variable to iterate over and index match with diskstats's
	// content order MUST match the columns in the diskstats file
	metrics []Metric
}

func NewDiskStatsCollector() DiskStatsCollector {
	return DiskStatsCollector{
		info: Metric{
			desc: prometheus.NewDesc(prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "info"),
				"Info of Ondat volumes and devices.",
				append(pvcLabels, "device", "major", "minor"), nil,
			),
			valueType: prometheus.GaugeValue,
		},
		metrics: []Metric{
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "reads_completed_total"),
					"The total number of reads completed successfully.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "reads_merged_total"),
					"The total number of reads merged.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "read_bytes_total"),
					"The total number of bytes read successfully.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "read_time_seconds_total"),
					"The total number of seconds spent by all reads.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "writes_completed_total"),
					"The total number of writes completed successfully.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "writes_merged_total"),
					"The number of writes merged.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "written_bytes_total"),
					"The total number of bytes written successfully.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "write_time_seconds_total"),
					"This is the total number of seconds spent by all writes.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "io_now"),
					"The number of I/Os currently in progress.",
					pvcLabels, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "io_time_seconds_total"),
					"Total seconds spent doing I/Os.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "io_time_weighted_seconds_total"),
					"The weighted # of seconds spent doing I/Os.",
					pvcLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "discards_completed_total"),
					"The total number of discards completed successfully.",
					pvcLabels, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "discards_merged_total"),
					"The total number of discards merged.",
					pvcLabels, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "discarded_sectors_total"),
					"The total number of sectors discarded successfully.",
					pvcLabels, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "discard_time_seconds_total"),
					"This is the total number of seconds spent by all discards.",
					pvcLabels, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "flush_requests_total"),
					"The total number of flush requests completed successfully",
					pvcLabels, nil,
				), valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "flush_requests_time_seconds_total"),
					"This is the total number of seconds spent by all flush requests.",
					pvcLabels, nil,
				), valueType: prometheus.CounterValue,
			},
		},
	}
}

func (c DiskStatsCollector) Name() string {
	return DISKSTATS_COLLECTOR_NAME
}

func (c DiskStatsCollector) Collect(log *zap.SugaredLogger, ch chan<- prometheus.Metric, ondatVolumes []*Volume) error {
	log.Debug("starting diskstats metrics collector")
	log = log.With("collector", DISKSTATS_COLLECTOR_NAME)

	if len(ondatVolumes) == 0 {
		log.Debug("no Ondat volumes, metrics collector finished early")
		return nil
	}

	err := ExtractOndatVolumesNumbers(ondatVolumes)
	if err != nil {
		log.Errorw("error getting Ondat volumes major and minor numbers", "error", err)
		return err
	}

	diskstats, err := ProcDiskstats()
	if err != nil {
		log.Errorw("error reading diskstats", "error", err)
		return err
	}

	for _, localVol := range ondatVolumes {
		logScope := log.With("pvc", localVol.Labels.PVC, "pvc_namespace", localVol.Labels.PVCNamespace)

		for _, stats := range diskstats {
			// match with Ondat volume through diskstat row's Major and Minor numbers
			if localVol.Major != int(stats.MajorNumber) || localVol.Minor != int(stats.MinorNumber) {
				continue
			}

			// Build the info metric for each diskstate line (volume) processed.
			// Its value is not relevant as we only care about the labels.
			// Failure to do so shouldn't stop us from collecting any further metrics.
			metric, err := prometheus.NewConstMetric(c.info.desc, c.info.valueType, 1.0, localVol.Labels.PVC, localVol.Labels.PVCNamespace, stats.DeviceName, fmt.Sprint(localVol.Major), fmt.Sprint(localVol.Minor))
			if err != nil {
				logScope.Errorw("encountered error while building metric", "metric", c.info.desc.String(), "error", err)
			} else {
				ch <- metric
			}

			diskSectorSize := 512.0
			logicalBlockSize, err := GetBlockDeviceLogicalBlockSize(stats.DeviceName)
			if err != nil {
				logScope.Errorw("error reading device logical block size, falling back to default", "error", err)
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
					// Didn't read all the above fields from diskstats.
					// Kernel version must be lower than v5.5 where these
					// fields don't exist yet.
					log.Debugf("diskstats number of colums processed was %s. If on kernel older than v5.5 this msg can be ignored.")
					break
				}

				metric, err := prometheus.NewConstMetric(c.metrics[i].desc, c.metrics[i].valueType, val, localVol.Labels.PVC, localVol.Labels.PVCNamespace)
				if err != nil {
					logScope.Errorw("encountered error while building metric", "metric", c.metrics[i].desc.String(), "error", err)
					continue
				}
				ch <- metric
			}
		}
	}

	log.Debug("finished metrics collector")
	return nil
}
