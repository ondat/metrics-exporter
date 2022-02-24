package main

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// DiskStatsCollector implements the prometheus Collector interface
type DiskStatsCollector struct {
	// TODO add logger

	// info metrics of all the scraped PVCs
	infoDesc Metric
	// all other metrics we gather from diskstats
	// usefull as a standalone variable to iterate over and index match with diskstats's content
	// order must match the columns in the diskstats file
	descs []Metric
}

func NewDiskStatsCollector() DiskStatsCollector {
	return DiskStatsCollector{
		infoDesc: Metric{
			desc: prometheus.NewDesc(prometheus.BuildFQName(ONDAT_NAMESPACE, DISK_SUBSYSTEM, "info"),
				"Info of Ondat volumes and matching devices.",
				[]string{"device", "pvc", "major", "minor"},
				nil,
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

func (c DiskStatsCollector) Collect(ch chan<- prometheus.Metric) {
	timer := time.Now()

	if err := ValidateDir(STOS_VOLUMES_PATH); err != nil {
		// TODO handle error
		// can return early
	}

	volumes, err := GetOndatVolumes()
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
		// can return early
	}

	diskstats, err := ProcDiskstats()
	if err != nil {
		fmt.Printf("err: %s\n", err.Error())
	}

	for _, vol := range volumes {
		// populate vol obj with what we want from the CP state files on this node
		err = GetOndatVolumeMount(vol)
		if err != nil {
			fmt.Printf("err: %s\n", err.Error())
			continue
		}

		for _, stats := range diskstats {
			// match Ondat volume with diskstat row's Major and Minor numbers
			if vol.Major != int(stats.MajorNumber) || vol.Minor != int(stats.MinorNumber) {
				continue
			}
			vol.metrics = stats

			ch <- NewConstMetric(c.infoDesc.desc, c.infoDesc.valueType, 1.0, stats.DeviceName, vol.PVC, fmt.Sprint(vol.Major), fmt.Sprint(vol.Minor))

			diskSectorSize := 512.0
			logicalBlockSize, err := GetBlockDeviceLogicalBlockSize(stats.DeviceName)
			if err != nil {
				fmt.Printf("err: %s\n", err.Error())
			} else {
				diskSectorSize = float64(logicalBlockSize)
				fmt.Printf("changing disck sector size to %f\n", diskSectorSize)
			}

			statCount := stats.IoStatsCount - 3 // Total diskstats record count, less MajorNumber, MinorNumber and DeviceName

			for i, val := range []float64{
				float64(stats.ReadIOs),
				float64(stats.ReadMerges),
				float64(stats.ReadSectors) * diskSectorSize,
				float64(stats.ReadTicks) * SECONDS_PER_TICK,
				float64(stats.WriteIOs),
				float64(stats.WriteMerges),
				float64(stats.WriteSectors) * diskSectorSize,
				float64(stats.WriteTicks) * SECONDS_PER_TICK,
				float64(stats.IOsInProgress),
				float64(stats.IOsTotalTicks) * SECONDS_PER_TICK,
				float64(stats.WeightedIOTicks) * SECONDS_PER_TICK,
				float64(stats.DiscardIOs),
				float64(stats.DiscardMerges),
				float64(stats.DiscardSectors),
				float64(stats.DiscardTicks) * SECONDS_PER_TICK,
				float64(stats.FlushRequestsCompleted),
				float64(stats.TimeSpentFlushing) * SECONDS_PER_TICK,
			} {
				if i >= statCount {
					break
				}

				ch <- NewConstMetric(c.descs[i].desc, c.descs[i].valueType, val, vol.PVC)
			}
		}
	}

	// if success/failure and duration of the scrape
	ch <- NewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(timer).Seconds(), "diskstats")
	// TODO push "0" on any failures?
	ch <- NewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, 1, "diskstats")
}

func NewConstMetric(desc *prometheus.Desc, valueType prometheus.ValueType, value float64, labels ...string) prometheus.Metric {
	m, err := prometheus.NewConstMetric(desc, valueType, value, labels...)
	if err != nil {
		// handle error?
		fmt.Printf("failed to create new ConstMetric. Metric desc: %s. Value: %f. Labels: %s.\n", desc.String(), value, labels[:])
	}
	return m
}
