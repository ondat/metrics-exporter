package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

var stuckMounts = make(map[string]struct{})
var stuckMountsMtx = &sync.Mutex{}

type filesystemLabels struct {
	device, mountPoint, fsType, options string
}

type FileSystemCollector struct {
	log logr.Logger

	descs []Metric
}

func NewFileSystemCollector(log logr.Logger) FileSystemCollector {
	return FileSystemCollector{
		log: log,
		descs: []Metric{
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, FILE_SYSTEM_SUBSYSTEM, "size_bytes"),
					"Filesystem size in bytes.",
					fsLabels, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, FILE_SYSTEM_SUBSYSTEM, "free_bytes"),
					"Filesystem free space in bytes.",
					fsLabels, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, FILE_SYSTEM_SUBSYSTEM, "avail_bytes"),
					"Filesystem space available to non-root users in bytes.",
					fsLabels, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, FILE_SYSTEM_SUBSYSTEM, "files"),
					"Filesystem total file nodes.",
					fsLabels, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, FILE_SYSTEM_SUBSYSTEM, "files_free"),
					"Filesystem total free file nodes.",
					fsLabels, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(ONDAT_NAMESPACE, FILE_SYSTEM_SUBSYSTEM, "readonly"),
					"Filesystem read-only status.",
					fsLabels, nil,
				),
				valueType: prometheus.GaugeValue,
			},
		},
	}
}

func (c FileSystemCollector) Describe(ch chan<- *prometheus.Desc) {
	// ch <- scrapeDurationMetric.desc
	ch <- scrapeSuccessMetric.desc
}

func (c FileSystemCollector) Collect(ch chan<- prometheus.Metric) {
	c.log.Info("Starting fs metrics collector")
	count := 0

	timeStart := time.Now()

	// All Ondat volumes fetched from the storageos container's API
	// Cluster wide thus only one request is needed
	// TODO both collectors are fetching the same data from the API
	// this needs work
	ondatVolumes, err := GetOndatVolumesAPI(c.log, apiSecretsPath)
	if err != nil {
		c.log.Error(err, "error contacting Ondat API")
		ReportScrapeResult(c.log, ch, timeStart, "filesystem", false)
		return
	}

	// TODO consider skipping getting all fs mounted devices
	// and fetch the data for each Ondat volume directly
	mps, err := mountPointDetails(c.log)
	if err != nil {
		c.log.Info("error mountPointDetails")
		return
	}

	for _, labels := range mps {
		if !strings.HasPrefix(labels.device, "/var/lib/storageos/volumes") {
			// level.Debug(c.logger).Log("msg", "Ignoring mount point", "mountpoint", labels.mountPoint)
			continue
		}

		c.log.Info("processing mount", "device", labels.device, "mountPoint", labels.mountPoint)

		stuckMountsMtx.Lock()
		if _, ok := stuckMounts[labels.mountPoint]; ok {
			ReportScrapeResult(c.log, ch, timeStart, "filesystem", false)
			// TODO device error metric
			c.log.Info("error mount stuck")

			// level.Debug(c.logger).Log("msg", "Mount point is in an unresponsive state", "mountpoint", labels.mountPoint)
			stuckMountsMtx.Unlock()
			continue
		}
		stuckMountsMtx.Unlock()

		// The success channel is used do tell the "watcher" that the stat
		// finished successfully. The channel is closed on success.
		success := make(chan struct{})
		go stuckMountWatcher(labels.mountPoint, success, c.log)

		buf := new(unix.Statfs_t)
		err = unix.Statfs(labels.mountPoint, buf)
		stuckMountsMtx.Lock()
		close(success)
		// If the mount has been marked as stuck, unmark it and log it's recovery.
		if _, ok := stuckMounts[labels.mountPoint]; ok {
			// level.Debug(c.logger).Log("msg", "Mount point has recovered, monitoring will resume", "mountpoint", labels.mountPoint)
			delete(stuckMounts, labels.mountPoint)
		}
		stuckMountsMtx.Unlock()

		if err != nil {
			ReportScrapeResult(c.log, ch, timeStart, "filesystem", false)
			// TODO device error metric
			c.log.Error(err, "error parsing fs stats into buffer", "device", labels.device, "mountPoint", labels.mountPoint)

			// level.Debug(c.logger).Log("msg", "Error on statfs() system call", "rootfs", rootfsFilePath(labels.mountPoint), "err", err)
			continue
		}

		var ro float64
		for _, option := range strings.Split(labels.options, ",") {
			if option == "ro" {
				ro = 1
				break
			}
		}

		// extract the volume ID from the mount
		// format: /var/lib/storageos/volumes/v.06115715-2901-49d4-9a05-fd4641b82d6d
		tmp := strings.Split(labels.device, "/")
		volID := strings.TrimPrefix(tmp[len(tmp)-1], "v.")

		var pvc string
		for _, vol := range ondatVolumes {
			if vol.ID == volID {
				pvc = vol.PVC
				break
			}
		}

		// TODO labels
		for i, val := range []float64{
			float64(buf.Blocks) * float64(buf.Bsize), // blocks * size per block = total space (bytes)
			float64(buf.Bfree) * float64(buf.Bsize),  // available blocks * size per block = total free space (bytes)
			float64(buf.Bavail) * float64(buf.Bsize), // available blocks * size per block = free space to non-root users (bytes)
			float64(buf.Files),                       // total inodes
			float64(buf.Ffree),                       // total free inodes
			ro,
		} {
			m := Metric{desc: c.descs[i].desc, valueType: c.descs[i].valueType}
			ch <- NewConstMetric(c.log, m, val, pvc, labels.device, labels.fsType, labels.mountPoint)
		}
		count++
	}
	c.log.Info("finished fs metrics colletor", "mounts read", count)
	ReportScrapeResult(c.log, ch, timeStart, "filesystem", true)
}

// stuckMountWatcher listens on the given success channel and if the channel closes
// then the watcher does nothing. If instead the timeout is reached, the
// mount point that is being watched is marked as stuck.
func stuckMountWatcher(mountPoint string, success chan struct{}, logger logr.Logger) {
	mountCheckTimer := time.NewTimer(time.Second * 5)
	defer mountCheckTimer.Stop()
	select {
	case <-success:
		// Success
	case <-mountCheckTimer.C:
		// Timed out, mark mount as stuck
		stuckMountsMtx.Lock()
		select {
		case <-success:
			// Success came in just after the timeout was reached, don't label the mount as stuck
		default:
			// level.Debug(logger).Log("msg", "Mount point timed out, it is being labeled as stuck and will not be monitored", "mountpoint", mountPoint)
			stuckMounts[mountPoint] = struct{}{}
		}
		stuckMountsMtx.Unlock()
	}
}

func mountPointDetails(logger logr.Logger) ([]filesystemLabels, error) {
	file, err := os.Open("/proc/1/mounts")
	if errors.Is(err, os.ErrNotExist) {
		// Fallback to `/proc/mounts` if `/proc/1/mounts` is missing due hidepid.
		// level.Debug(logger).Log("msg", "Reading root mounts failed, falling back to system mounts", "err", err)
		file, err = os.Open("/proc/mounts")
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseFilesystemLabels(file)
}

func parseFilesystemLabels(r io.Reader) ([]filesystemLabels, error) {
	var filesystems []filesystemLabels

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())

		if len(parts) < 4 {
			return nil, fmt.Errorf("malformed mount point information: %q", scanner.Text())
		}

		// Ensure we handle the translation of \040 and \011
		// as per fstab(5).
		parts[1] = strings.Replace(parts[1], "\\040", " ", -1)
		parts[1] = strings.Replace(parts[1], "\\011", "\t", -1)

		filesystems = append(filesystems, filesystemLabels{
			device:     parts[0],
			mountPoint: parts[1],
			fsType:     parts[2],
			options:    parts[3],
		})
	}

	return filesystems, scanner.Err()
}
