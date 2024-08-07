package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tvrzna/go-utils/args"
)

const (
	pathDiskstats = "/proc/diskstats"
)

var buildVersion string

type context struct {
	devices     map[string]*device
	idlePeriod  int
	gracePeriod int
	threshold   int
	verbose     bool
}

// Initializes the context
func initContext(arg []string) *context {
	c := &context{idlePeriod: 300, gracePeriod: 600, threshold: 1}

	c.devices = make(map[string]*device)

	args.ParseArgs(arg, func(arg, value string) {
		switch arg {
		case "-h", "--help":
			fmt.Printf(`Usage: qd [options] [device ...]
Options:
	-h, --help			print this help
	-v, --version			print version
	-i, --idle [SECONDS]		sets idle period, before device is put into standby mode (default = 300)
	-g, --grace [SECONDS]		sets grace period, before device could be put into standby mode after return from standby mode (default = 600)
	-t, --treshold [IOPS]		sets IOPS treshold (default = 1)
	-V, --verbose			adds verbosity into logs
`)
			os.Exit(0)
		case "-v", "--version":
			fmt.Printf("quietdisk %s\nhttps://github.com/tvrzna/quietdisk\n\nReleased under the MIT License.\n", c.getVersion())
			os.Exit(0)
		case "-i", "--idle":
			c.idlePeriod, _ = strconv.Atoi(value)
		case "-g", "--grace":
			c.gracePeriod, _ = strconv.Atoi(value)
		case "-l", "--list":
			// TODO: list all available devices
		case "-V", "--verbose":
			c.verbose = true
		default:
			// TODO: support * for all devices
			if strings.TrimSpace(value) != "" {
				c.devices[strings.TrimSpace(value)] = nil
			}
		}
	})

	if len(c.devices) == 0 {
		log.Fatal("no device is defined, check help.")
	}

	c.initDevices()

	return c
}

// Starts context and its periodical checks
func (c *context) start() {
	log.Print("quietdisk started")

	sleep := 60
	if sleep > c.idlePeriod {
		sleep = c.idlePeriod
	}

	for {
		if c.verbose {
			log.Print("updating devices")
		}
		c.updateDevices()

		time.Sleep(time.Duration(sleep) * time.Second)
	}
}

// Performs initialization of devices
func (c *context) initDevices() {
	for id := range c.devices {
		dev, err := initDevice(id)
		if err != nil || dev == nil {
			log.Print(err)
			continue
		}
		c.devices[id] = dev
	}
}

// Performs updates on each device, if is available. Devices not listed in /proc/diskstats or partitions are skipped.
func (c *context) updateDevices() {
	for _, d := range c.devices {
		d.updateMajorMinor()
	}

	b, err := os.ReadFile(pathDiskstats)
	if err != nil {
		log.Print(err)
	}
	diskstats := string(b)
	scanner := bufio.NewScanner(strings.NewReader(diskstats))
	for scanner.Scan() {
		data := strings.Fields(scanner.Text())

		major, _ := strconv.Atoi(data[0])
		minor, _ := strconv.Atoi(data[1])
		name := data[2]
		rIops, _ := strconv.ParseUint(data[3], 10, 64)
		wIops, _ := strconv.ParseUint(data[7], 10, 64)

		dev := c.getDevice(major, minor)
		if dev == nil || dev.isPartition() {
			continue
		}
		dev.name = name

		now := time.Now().Unix()

		// Check for changes on read and write IOPS
		if rIops >= dev.rIops+uint64(c.threshold) || wIops >= dev.wIops+uint64(c.threshold) {
			dev.rIops, dev.wIops, dev.lastChange, dev.isSleeping = rIops, wIops, now, false
			if c.verbose {
				log.Printf("rIops or wIops has changed on '%s'", dev.device)
			}
			continue
		}

		// If is not sleeping and should have, put device to sleep
		if !dev.isSleeping && dev.lastChange+int64(c.idlePeriod) <= now {
			if dev.lastStandBy == 0 || dev.lastStandBy+int64(c.gracePeriod) <= now {
				dev.lastStandBy, dev.isSleeping = now, true
				log.Printf("going to put '%s' to sleep ", dev.device)
				if err := putDriveToSleep(dev.device); err != nil {
					log.Print(err)
				}
			} else {
				if c.verbose {
					log.Printf("it is too soon to put '%s' to sleep", dev.device)
				}
			}
		}
	}
}

// Gets device from map by major and minor identificators
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
