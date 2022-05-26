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

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

const FILE_SYSTEM_COLLECTOR_NAME = "filesystem"

var stuckMounts = make(map[string]struct{})
var stuckMountsMtx = &sync.Mutex{}

type filesystemLabels struct {
	device, mountPoint, fsType, options string
}

type FileSystemCollector struct {
	deviceErrors Metric

	metrics []Metric
}

func NewFileSystemCollector() FileSystemCollector {
	return FileSystemCollector{
		deviceErrors: Metric{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(ONDAT_NAMESPACE, FILE_SYSTEM_SUBSYSTEM, "device_error"),
				"Whether an error occurred while getting statistics for the given device.",
				fsLabels, nil,
			),
			valueType: prometheus.GaugeValue,
		},
		metrics: []Metric{
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

func (c FileSystemCollector) Name() string {
	return FILE_SYSTEM_COLLECTOR_NAME
}

func (c FileSystemCollector) Collect(log *zap.SugaredLogger, ch chan<- prometheus.Metric, ondatVolumes []VolumePVC) error {
	log.Debug("starting filesystem metrics collector")
	log = log.With("collector", FILE_SYSTEM_COLLECTOR_NAME)

	if len(ondatVolumes) == 0 {
		log.Debug("no Ondat volumes, metrics collector finished early")
		return nil
	}

	// TODO consider skipping getting all fs mounted devices
	// and fetch the data for each Ondat volume directly
	mps, err := mountPointDetails(log)
	if err != nil {
		log.Errorw("failed to read mounts", "error", err)
		return err
	}

	for _, labels := range mps {
		if !strings.HasPrefix(labels.device, "/var/lib/storageos/volumes") {
			continue
		}

		// extract the volume ID from the mount
		// format: /var/lib/storageos/volumes/v.06115715-2901-49d4-9a05-fd4641b82d6d
		tmp := strings.Split(labels.device, "/")
		volID := strings.TrimPrefix(tmp[len(tmp)-1], "v.")

		var pvc, pvcNamespace string
		for _, vol := range ondatVolumes {
			if vol.ID == volID {
				pvc = vol.PVC
				pvcNamespace = vol.Namespace
				break
			}
		}

		logScope := log.With("pvc", pvc, "pvc_namespace", pvcNamespace, "device", labels.device, "mountpoint", labels.mountPoint)

		stuckMountsMtx.Lock()
		if _, ok := stuckMounts[labels.mountPoint]; ok {
			logScope.Errorw("mount point is in an unresponsible state", "mountpoint", labels.mountPoint)
			metric, err := prometheus.NewConstMetric(c.deviceErrors.desc, c.deviceErrors.valueType, 1, pvc, pvcNamespace, labels.device, labels.fsType, labels.mountPoint)
			if err != nil {
				logScope.Errorw("encountered error while building metric", "metric", c.deviceErrors.desc.String(), "error", err)
				continue
			}
			ch <- metric

			stuckMountsMtx.Unlock()
			continue
		}
		stuckMountsMtx.Unlock()

		// The success channel is used do tell the "watcher" that the stat
		// finished successfully. The channel is closed on success.
		success := make(chan struct{})
		go stuckMountWatcher(log, labels.mountPoint, success, log)

		buf := new(unix.Statfs_t)
		err = unix.Statfs(labels.mountPoint, buf)
		stuckMountsMtx.Lock()
		close(success)
		// If the mount has been marked as stuck, unmark it and log it's recovery.
		if _, ok := stuckMounts[labels.mountPoint]; ok {
			logScope.Debugw("mount point has recovered, monitoring will resume", "mountpoint", labels.mountPoint)
			delete(stuckMounts, labels.mountPoint)
		}
		stuckMountsMtx.Unlock()

		if err != nil {
			logScope.Errorw("error on statfs() system call", "device", labels.device, "mountpoint", labels.mountPoint, "error", err)
			metric, err := prometheus.NewConstMetric(c.deviceErrors.desc, c.deviceErrors.valueType, 1, pvc, pvcNamespace, labels.device, labels.fsType, labels.mountPoint)
			if err != nil {
				logScope.Errorw("encountered error while building metric", "metric", c.deviceErrors.desc.String(), "error", err)
			} else {
				ch <- metric
			}
			continue
		}

		var ro float64
		for _, option := range strings.Split(labels.options, ",") {
			if option == "ro" {
				ro = 1
				break
			}
		}

		for i, val := range []float64{
			float64(buf.Blocks) * float64(buf.Bsize), // blocks * size per block = total space (bytes)
			float64(buf.Bfree) * float64(buf.Bsize),  // available blocks * size per block = total free space (bytes)
			float64(buf.Bavail) * float64(buf.Bsize), // available blocks * size per block = free space to non-root users (bytes)
			float64(buf.Files),                       // total inodes
			float64(buf.Ffree),                       // total free inodes
			ro,
		} {
			metric, err := prometheus.NewConstMetric(c.metrics[i].desc, c.metrics[i].valueType, val, pvc, pvcNamespace, labels.device, labels.fsType, labels.mountPoint)
			if err != nil {
				logScope.Errorw("encountered error while building metric", "metric", c.metrics[i].desc.String(), "error", err)
				continue
			}
			ch <- metric
		}
		metric, err := prometheus.NewConstMetric(c.deviceErrors.desc, c.deviceErrors.valueType, 0, pvc, pvcNamespace, labels.device, labels.fsType, labels.mountPoint)
		if err != nil {
			logScope.Errorw("encountered error while building metric", "metric", c.deviceErrors.desc.String(), "error", err)
			continue
		}
		ch <- metric
	}

	log.Debug("finished metrics collector")
	return nil
}

// stuckMountWatcher listens on the given success channel and if the channel closes
// then the watcher does nothing. If instead the timeout is reached, the
// mount point that is being watched is marked as stuck.
func stuckMountWatcher(log *zap.SugaredLogger, mountPoint string, success chan struct{}, logger *zap.SugaredLogger) {
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
			log.Errorw("mount point timed out, it is being labeled as stuck and will not be monitored", "mountpoint", mountPoint)
			stuckMounts[mountPoint] = struct{}{}
		}
		stuckMountsMtx.Unlock()
	}
}

func mountPointDetails(logger *zap.SugaredLogger) ([]filesystemLabels, error) {
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
