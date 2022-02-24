package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/procfs/blockdevice"
)

const (
	STOS_VOLUMES_PATH   = "/var/lib/storageos/volumes"
	PROC_DISKSTATS_PATH = "/proc/diskstats"
)

type Labels struct {
	PvName  string `json:"csi.storage.k8s.io/pv/name"`
	PvcName string `json:"csi.storage.k8s.io/pvc/name"`
}

type VolumeJson struct {
	Labels `json:"labels"`
}

type Volume struct {
	Major int
	Minor int
	ID    string // CP ID
	PVC   string // K8s PVC name, friendly format (not the ID)

	// metrics
	// reusing prometheus struct
	metrics blockdevice.Diskstats
}

// ValidateDir checks if the path can be read and is a Directory
func ValidateDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("could not read dir %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", dir)
	}
	return nil
}

// ProcDiskstats reads the diskstats file and returns
// an array of Diskstats (one per line/device)
func ProcDiskstats() ([]blockdevice.Diskstats, error) {
	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	diskstats := []blockdevice.Diskstats{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		d := &blockdevice.Diskstats{}
		d.IoStatsCount, err = fmt.Sscanf(scanner.Text(), "%d %d %s %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
			&d.MajorNumber,
			&d.MinorNumber,
			&d.DeviceName,
			&d.ReadIOs,
			&d.ReadMerges,
			&d.ReadSectors,
			&d.ReadTicks,
			&d.WriteIOs,
			&d.WriteMerges,
			&d.WriteSectors,
			&d.WriteTicks,
			&d.IOsInProgress,
			&d.IOsTotalTicks,
			&d.WeightedIOTicks,
			&d.DiscardIOs,
			&d.DiscardMerges,
			&d.DiscardSectors,
			&d.DiscardTicks,
			&d.FlushRequestsCompleted,
			&d.TimeSpentFlushing,
		)
		// The io.EOF error can be safely ignored because it just means we read fewer than
		// the full 20 fields. Happens on kernel versions lower than 5.5
		if err != nil && err != io.EOF {
			return diskstats, err
		}
		if d.IoStatsCount >= 14 {
			diskstats = append(diskstats, *d)
		}
	}
	return diskstats, scanner.Err()
}

func GetBlockDeviceLogicalBlockSize(device string) (uint64, error) {
	data, err := ioutil.ReadFile("/sys/block/queue/" + device + "/logical_block_size")
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func GetOndatVolumes() ([]*Volume, error) {
	fmt.Printf("processing /var/lib/storageos/volumes\n") // we don't want this sort of message on the client's side

	output, err := exec.Command("ls", "-l", "/var/lib/storageos/volumes").Output()
	if err != nil {
		fmt.Printf("error running ls\n")
		return nil, err
	}
	out := strings.Split(string(output), "\n")
	// exclude first and last elements
	// first line of `ls -l`` shows the total size of blocks on that
	// dir and the ending "\n" creates an empty element on the array
	out = out[1 : len(out)-1]
	if len(out) == 0 {
		return nil, fmt.Errorf("no stos volumes")
	}

	volumes := []*Volume{}

	for _, line := range out {
		// fmt.Printf("processing line: %s\n", line)

		// don't care about anything other than block devices
		if line[0] != 'b' {
			continue
		}

		var (
			// discard is used as a filler for the columns in the output of "ls -l"
			// that we don't care about
			discard string
			// deviceName is in the format "v.<uuid>" where the uuid represents
			// the volume ID in ControlPlane
			deviceName   string
			major, minor int
		)

		_, err = fmt.Sscanf(line,
			"%s %s %s %s %d, %d %s %s %s %s",
			&discard, &discard, &discard, &discard, &major, &minor, &discard, &discard, &discard, &deviceName,
		)
		if err != nil {
			// handle error
			return nil, fmt.Errorf("failed to ingest ls output")
		}

		volumes = append(volumes,
			&Volume{
				ID:    strings.Split(deviceName, ".")[1],
				Major: major,
				Minor: minor,
			},
		)
	}

	return volumes, nil
}

// must be called after GetOndatVolumes
func GetOndatVolumeMount(vol *Volume) error {
	fmt.Printf("processing /var/lib/storageos/state/v.%s.json\n", vol.ID) // we don't want this sort of message on the client's side

	content, err := os.ReadFile("/var/lib/storageos/state/v." + vol.ID + ".json")
	if err != nil {
		return fmt.Errorf("failed to open file, err: %s", err.Error())
	}

	out := &VolumeJson{}
	err = json.Unmarshal(content, out)
	if err != nil {
		return fmt.Errorf("failed to unmarshal volume state file, err: %s", err.Error())
	}

	vol.PVC = out.PvcName
	return nil
}
