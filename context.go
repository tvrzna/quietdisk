package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tvrzna/go-utils/args"
)

type contextAction byte

const (
	contextActionDaemon contextAction = iota
	contextActionList
	contextActionCheck
	contextActionSleep
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
	hddOnly     bool
	action      contextAction
	d           *daemon
}

// Initializes the context
func initContext(osArgs []string) *context {
	c := &context{idlePeriod: 300, gracePeriod: 600, threshold: 1, devices: make(map[string]*device), action: contextActionDaemon}

	osArgs = osArgs[1:]
	args.ParseArgs(osArgs, func(arg, value string) {
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
			c.action = contextActionList
		case "-C", "-c", "--check":
			c.action = contextActionCheck
		case "-Y", "--sleep":
			c.action = contextActionSleep
		case "-V", "--verbose":
			c.verbose = true
		case "-H", "--hdd-only":
			c.hddOnly = true
		default:
			val := strings.TrimSpace(arg)
			if val == "all" {
				c.allDevices = true
			} else if strings.HasPrefix(val, "/") {
				c.devices[val] = nil
			}
		}
	})

	if c.action == contextActionDaemon {
		c.d = &daemon{c}
	}

	return c
}

// Starts the daemon.
func (c *context) startDaemon() {
	c.d.start()
}

// Lists all devices, exclude loops and zero size devices.
func (c *context) listAllDevices() []string {
	var result []string

	dir, err := os.ReadDir(pathBlocks)
	if err != nil {
		c.logError(fmt.Sprintf("could not read from '%s'", pathBlocks))
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
		result = append(result, devName)
	}

	return result
}

// Performs initialization of devices
func (c *context) initDevices() error {
	if c.allDevices {
		c.devices = make(map[string]*device)
		if c.action != contextActionDaemon {
			for _, dev := range c.listAllDevices() {
				c.devices[dev] = nil
			}
		}
	}
	if len(c.devices) == 0 && !c.allDevices {
		return errors.New("no device is defined")
	}

	for id, dev := range c.devices {
		if dev != nil {
			continue
		}

		dev, err := dev.initDevice(id, c.hddOnly)
		if err != nil || dev == nil {
			if err != nil {
				c.logError(err)
			}
			delete(c.devices, id)
			continue
		}
		c.devices[id] = dev
	}

	return nil
}

// Prints listed devices with their current power mode/state.
func (c *context) printListedDevices() {
	c.allDevices = true
	c.checkDevices()
}

// Checks power state of listed devices.
func (c *context) checkDevices() {
	if err := c.initDevices(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, dev := range c.devices {
		state, err := dev.getDriveState()
		fmt.Printf("%s (%s)", dev.device, state.stringify())
		if err != nil {
			fmt.Printf(": %v", err)
		}
		fmt.Printf("\n")
	}
}

// Puts listed devices into sleep/standby mode.
func (c *context) sleepDevices() {
	if err := c.initDevices(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, dev := range c.devices {
		err := dev.putDriveToSleep()
		fmt.Print(dev.device)
		if err != nil {
			fmt.Printf(": %v\n", err)
		} else {
			fmt.Printf(": putting into sleep\n")
		}
	}
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
	-H, --hdd-only			works only with HDDs (rotational drives), skipping SSDs and NVMe devices.
	-l, --list			lists all available devices with their power mode
	-C, -c, --check			check power mode of listed devices
	-Y, --sleep			put listed devices into sleep mode
	-i, --idle [SECONDS]		sets idle period, before device is put into standby mode (default = 300)
	-g, --grace [SECONDS]		sets grace period, before device could be put into standby mode after return from standby mode (default = 600)
	-t, --treshold [IOPS]		sets IOPS treshold (default = 1)
	-V, --verbose			adds verbosity into logs
`)
	os.Exit(0)
}

// Logs error in case of daemon, in case of CLI it just prints the error.
func (c *context) logError(a ...any) {
	if c.action == contextActionDaemon {
		log.Print(a...)
	} else {
		fmt.Println(a...)
	}
}
