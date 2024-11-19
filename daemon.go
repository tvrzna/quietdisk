package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type daemon struct {
	c *context
}

// Starts daemon and its periodical checks
func (d *daemon) start() {
	if err := d.c.initDevices(); err != nil {
		log.Fatal(err)
	}

	log.Print("quietdisk started")

	sleep := 60
	if sleep > d.c.idlePeriod {
		sleep = d.c.idlePeriod
	}

	for {
		if d.c.verbose {
			log.Print("updating devices")
		}
		d.updateDevices()

		time.Sleep(time.Duration(sleep) * time.Second)
	}
}

// Performs updates on each device, if is available. Devices not listed in /proc/diskstats or partitions are skipped.
func (d *daemon) updateDevices() {
	if err := d.refreshDevices(); err != nil {
		log.Print(err)
	}

	b, err := os.ReadFile(pathDiskstats)
	if err != nil {
		log.Print(err)
	}
	diskstats := string(b)
	scanner := bufio.NewScanner(strings.NewReader(diskstats))
	for scanner.Scan() {
		data := strings.Fields(scanner.Text())

		if len(data) < 8 {
			log.Printf("incorrect number of fields from %s", pathDiskstats)
			continue
		}

		major, _ := strconv.Atoi(data[0])
		minor, _ := strconv.Atoi(data[1])
		name := data[2]
		rIops, _ := strconv.ParseUint(data[3], 10, 64)
		wIops, _ := strconv.ParseUint(data[7], 10, 64)

		dev := d.getDevice(major, minor)
		if dev == nil {
			continue
		}
		dev.name = name

		now := time.Now().Unix()

		// Check for changes on read and write IOPS
		if rIops >= dev.rIops && rIops-dev.rIops >= uint64(d.c.threshold) || wIops >= dev.wIops && wIops-dev.wIops >= uint64(d.c.threshold) {
			dev.rIops, dev.wIops, dev.lastChange, dev.isSleeping = rIops, wIops, now, false
			if d.c.verbose {
				log.Printf("rIops or wIops has changed on '%s'", dev.device)
			}
			continue
		}

		// Check, if drive is really sleeping
		if dev.isSleeping && dev.lastStandBy+int64(d.c.idlePeriod) <= now {
			if isSleeping, err := dev.isDriveSleeping(); err != nil {
				log.Print(err)
			} else if !isSleeping {
				log.Printf("'%s' is awake, but should be asleep ", dev.device)
				dev.isSleeping = false
			}
		}

		// If is not sleeping and should have, put device to sleep
		if !dev.isSleeping && dev.lastChange+int64(d.c.idlePeriod) <= now {
			if dev.lastStandBy == 0 || dev.lastStandBy+int64(d.c.gracePeriod) <= now {
				dev.lastStandBy, dev.isSleeping = now, true
				log.Printf("going to put '%s' to sleep ", dev.device)
				if err := dev.putDriveToSleep(); err != nil {
					log.Print(err)
				}
			} else {
				if d.c.verbose {
					log.Printf("it is too soon to put '%s' to sleep", dev.device)
				}
			}
		}
	}
}

// Refreshes map of devices to be used.
func (d *daemon) refreshDevices() error {
	if d.c.allDevices {
		devices := d.c.listAllDevices()
		for _, id := range devices {
			if _, exists := d.c.devices[id]; !exists {
				var dev *device
				var err error
				dev, err = dev.initDevice(id, d.c.hddOnly)
				if err == nil && dev != nil {
					d.c.devices[id] = dev
				}
			}
		}
	}

	if len(d.c.devices) == 0 {
		return errors.New("no device is available")
	}

	for _, dev := range d.c.devices {
		err := dev.updateMajorMinor()
		if err != nil {
			if d.c.allDevices {
				delete(d.c.devices, dev.device)
			}
		}
	}

	return nil
}

// Gets device from map by major and minor identificators.
func (d *daemon) getDevice(major, minor int) *device {
	for _, dev := range d.c.devices {
		if dev.exists && dev.major == major && dev.minor == minor {
			return dev
		}
	}
	return nil
}
