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
	// PROC_DISKSTATS_MIN_NUM_FIELDS is the minimum number of fields we expect
	// to find in the /proc/diskstats (kernels v4.18+ add more).
	// More about the /proc/diskstats format can be found here:
	// https://www.kernel.org/doc/Documentation/ABI/testing/procfs-diskstats
	PROC_DISKSTATS_MIN_NUM_FIELDS = 14
)

type Labels struct {
	PVC string `json:"csi.storage.k8s.io/pvc/name"`
}

// VolumeJson is used to extract the information we need from the CP state files
type VolumeJson struct {
	Labels `json:"labels"`
}

type Volume struct {
	Major int
	Minor int
	ID    string // Control Plane volume ID
	PVC   string // K8s friendly PVC name (not the ID)

	// metrics
	metrics blockdevice.Diskstats
}

// ValidateDir checks if the path can be read and is a Directory
func ValidateDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("could not read directory %q: %w", dir, err)
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
		if d.IoStatsCount >= PROC_DISKSTATS_MIN_NUM_FIELDS {
			diskstats = append(diskstats, *d)
		}
	}
	return diskstats, scanner.Err()
}

func GetBlockDeviceLogicalBlockSize(device string) (uint64, error) {
	data, err := ioutil.ReadFile("/sys/block/" + device + "/queue/logical_block_size")
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

// GetOndatVolumes parses the output from "ls -l" on the storageos block devices
// directory and builds a list of all volumes found.
// includes Major & Minor numbers and Volume ID
func GetOndatVolumes() ([]*Volume, error) {
	output, err := readOndatVolumes()
	if err != nil {
		return nil, err
	}

	return parseOndatVolumes(output)
}

func readOndatVolumes() ([]string, error) {
	outputRaw, err := exec.Command("ls", "-l", "/var/lib/storageos/volumes").Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	return strings.Split(string(outputRaw), "\n"), nil
}

func parseOndatVolumes(input []string) ([]*Volume, error) {
	// exclude first and last elements
	// first line of `ls -l`` shows the total size of blocks on that
	// dir and the ending "\n" creates an empty element on the array
	input = input[1 : len(input)-1]
	if len(input) == 0 {
		return []*Volume{}, nil
	}

	var (
		// discard is used as a filler for the columns in the output from
		// "ls -l" that we don't care about
		discard string
		// deviceName is in the format "v.<uuid>" where the uuid represents
		// the volume ID in ControlPlane
		deviceName   string
		major, minor int
		volumes      []*Volume = []*Volume{}
	)

	for _, line := range input {
		// only interested in block devices
		if line[0] != 'b' {
			continue
		}

		_, err := fmt.Sscanf(line,
			"%s %s %s %s %d, %d %s %s %s %s",
			&discard, &discard, &discard, &discard, &major, &minor, &discard, &discard, &discard, &deviceName,
		)
		if err != nil {
			// return nil, fmt.Errorf("error ingesting command output %s: %w", line, err)
			// TODO add logging and continue instead of failing
			continue
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

// GetOndatVolumeState parses a Control Plane volume state file and fills in all
// the data we are missing from the volume that is not found on OS files
func GetOndatVolumeState(vol *Volume) error {
	content, err := readOndatVolumeState(vol.ID)
	if err != nil {
		return err
	}

	return parseOndatVolumeState(vol, content)
}

func readOndatVolumeState(volID string) ([]byte, error) {
	file := fmt.Sprintf("/var/lib/storageos/state/v.%s.json", volID)

	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func parseOndatVolumeState(vol *Volume, content []byte) error {
	volumeState := &VolumeJson{}
	err := json.Unmarshal(content, volumeState)
	if err != nil {
		return fmt.Errorf("failed to parse file content: %w", err)
	}

	vol.PVC = volumeState.PVC
	return nil
}
