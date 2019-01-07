package main

import (
	"github.com/function61/james/pkg/shellmultipart"
	"math"
	"strconv"
	"strings"
)

const ramDetectScript = `free_output=$(free -m)

regex="Mem: +([0-9]+)"

if [[ ! $free_output =~ $regex ]]; then echo "Unable to resolve: ram_total_mb" 2>&1; exit 1; fi

ram_total_mb="${BASH_REMATCH[1]}"

echo "$ram_total_mb"`

const dockerVersionDetectScript = `docker_version_output=$(docker --version)

regex="version (.+)"

if [[ ! $docker_version_output =~ $regex ]]; then echo "Unable to resolve: docker_version" 2>&1; exit 1; fi

docker_version="${BASH_REMATCH[1]}"

echo "$docker_version"`

const diskTotalGigabytes = `df_output=$(df -h /)
# "/dev/sda1        39G  6.2G   33G  16% /"

regex=" +([0-9\.]+)G"

if [[ ! $df_output =~ $regex ]]; then echo "Unable to resolve: disk_total" 2>&1; exit 1; fi

disk_total="${BASH_REMATCH[1]}"

echo "$disk_total"`

func attachDetectors(scripts *shellmultipart.Multipart) func() (*NodeSpecs, error) {
	kernelVersion := scripts.AddPart("uname --kernel-release")
	osRelease := scripts.AddPart(`(source /etc/os-release; echo "$PRETTY_NAME")`)
	ramMb := scripts.AddPart(ramDetectScript)
	dockerVersion := scripts.AddPart(dockerVersionDetectScript)
	diskTotalGb := scripts.AddPart(diskTotalGigabytes)

	return func() (*NodeSpecs, error) {
		trimRightNewline := func(in string) string { return strings.TrimRight(in, "\n") }

		diskTotalGbParsed, err := strconv.ParseFloat(trimRightNewline(diskTotalGb.Output()), 64)
		if err != nil {
			return nil, err
		}

		ramMbParsed, err := strconv.ParseFloat(trimRightNewline(ramMb.Output()), 64)
		if err != nil {
			return nil, err
		}

		ramGb := math.Round(ramMbParsed/1024*10) / 10

		// remove uninteresting part in "Container Linux by CoreOS ...."
		osReleaseCut := strings.Replace(
			trimRightNewline(osRelease.Output()),
			"Container Linux by ",
			"",
			-1)

		return &NodeSpecs{
			KernelVersion: trimRightNewline(kernelVersion.Output()),
			OsRelease:     osReleaseCut,
			DockerVersion: trimRightNewline(dockerVersion.Output()),
			DiskGb:        diskTotalGbParsed,
			RamGb:         ramGb,
		}, nil
	}
}
