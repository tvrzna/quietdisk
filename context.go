package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tvrzna/go-utils/args"
)

const (
	pathDiskstats   = "/proc/diskstats"
	pathBlocks      = "/sys/block"
	pathClassBlocks = "/sys/class/block/"

	devicePrefix = "/dev/"
)

var buildVersion string

type context struct {
	devices     map[string]*device
	idlePeriod  int
	gracePeriod int
	threshold   int
	verbose     bool
	allDevices  bool
	d           *daemon
}

// Initializes the context
func initContext(arg []string) *context {
	c := &context{idlePeriod: 300, gracePeriod: 600, threshold: 1, devices: make(map[string]*device)}

	args.ParseArgs(arg, func(arg, value string) {
		switch arg {
		case "-h", "--help":
			c.printHelp()
		case "-v", "--version":
			fmt.Printf("quietdisk %s\nhttps://github.com/tvrzna/quietdisk\n\nReleased under the MIT License.\n", c.getVersion())
			os.Exit(0)
		case "-i", "--idle":
			c.idlePeriod, _ = strconv.Atoi(value)
		case "-g", "--grace":
			c.gracePeriod, _ = strconv.Atoi(value)
		case "-l", "--list":
			c.printListedDevices()
		case "-V", "--verbose":
			c.verbose = true
		default:
			val := strings.TrimSpace(value)
			if val == "*" {
				c.allDevices = true
			} else if val != "" {
				c.devices[strings.TrimSpace(value)] = nil
			}
		}
	})

	c.d = &daemon{c}

	return c
}

// Starts the daemon.
func (c *context) startDaemon() {
	c.d.start()
}

// Lists devices, exclude loops and zero size devices.
func (c *context) listDevices() map[string]*device {
	result := make(map[string]*device)

	dir, err := os.ReadDir(pathBlocks)
	if err != nil {
		log.Printf("could not read from '%s'", pathBlocks)
		return result
	}

	for _, f := range dir {
		if strings.HasPrefix(f.Name(), "loop") {
			continue
		}

		d, _ := os.ReadFile(filepath.Join(pathClassBlocks, f.Name(), "size"))
		size, _ := strconv.Atoi(strings.TrimSpace(string(d)))
		if size == 0 {
			continue
		}

		devName := filepath.Join(devicePrefix, f.Name())
		var dev *device
		result[devName], _ = dev.initDevice(devName)
	}

	return result
}

// Prints listed devices with their current power mode/state.
func (c *context) printListedDevices() {
	devices := c.listDevices()
	if len(devices) == 0 {
		fmt.Printf("No device to be listed")
		os.Exit(1)
	}
	fmt.Printf("Listed devices:\n")
	for _, dev := range devices {
		state, _ := dev.getDriveState()
		fmt.Printf("\t%s (%s)\n", dev.device, state.stringify())
	}
	os.Exit(0)
}

// Gets device from map by major and minor identificators.
func (c *context) getDevice(major, minor int) *device {
	for _, d := range c.devices {
		if d.exists && d.major == major && d.minor == minor {
			return d
		}
	}
	return nil
}

// Gets project version
func (c *context) getVersion() string {
	if buildVersion == "" {
		return "develop"
	}
	return buildVersion
}

// Prints help/usage
func (c *context) printHelp() {
	fmt.Printf(`Usage: qd [options] [device ...]
Options:
	-h, --help			print this help
	-v, --version			print version
	-l, --list			lists all available devices
	-i, --idle [SECONDS]		sets idle period, before device is put into standby mode (default = 300)
	-g, --grace [SECONDS]		sets grace period, before device could be put into standby mode after return from standby mode (default = 600)
	-t, --treshold [IOPS]		sets IOPS treshold (default = 1)
	-V, --verbose			adds verbosity into logs
`)
	os.Exit(0)
}
