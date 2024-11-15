package main

import (
	"bufio"
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
	if d.c.allDevices {
		d.c.devices = make(map[string]*device)
	}
	if len(d.c.devices) == 0 && !d.c.allDevices {
		log.Fatal("no device is defined, check help.")
	}
	d.initDevices()

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

// Prepares map of devices to be used.
func (d *daemon) prepareDevices() {
	if d.c.allDevices {
		devices := d.c.listDevices()
		for dev := range devices {
			if _, exists := d.c.devices[dev]; !exists {
				d.c.devices[dev] = devices[dev]
			}
		}
	}

	if len(d.c.devices) == 0 {
		log.Print("no device is available")
	}

	for _, dev := range d.c.devices {
		err := dev.updateMajorMinor()
		if err != nil {
			log.Print(err)
			if d.c.allDevices {
				delete(d.c.devices, dev.device)
			}
		}
	}
}

// Performs initialization of devices
func (d *daemon) initDevices() {
	for id, dev := range d.c.devices {
		if dev != nil {
			continue
		}

		dev, err := dev.initDevice(id)
		if err != nil || dev == nil {
			log.Print(err)
			delete(d.c.devices, id)
			continue
		}
		d.c.devices[id] = dev
	}
}

// Performs updates on each device, if is available. Devices not listed in /proc/diskstats or partitions are skipped.
func (d *daemon) updateDevices() {
	d.prepareDevices()

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

		dev := d.c.getDevice(major, minor)
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
