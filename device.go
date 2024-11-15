package main

import (
	"fmt"
	"os"
	"path"
	"syscall"
)

type device struct {
	device      string
	name        string
	major       int
	minor       int
	rIops       uint64
	wIops       uint64
	lastChange  int64
	lastStandBy int64
	exists      bool
	isSleeping  bool
}

type powerMode byte

const (
	powerModeStandby         powerMode = 0x00
	powerModeNvcacheSpindown powerMode = 0x40
	powerModeNvcache_spinup  powerMode = 0x41
	powerModeIdle            powerMode = 0x80
	powerModeActive          powerMode = 0xff
	unknown                  powerMode = 0xe0
)

// Initializes device
func (d *device) initDevice(dev string) (*device, error) {
	if d == nil {
		for dev[len(dev)-1] == '/' {
			dev = dev[:len(dev)-1]
		}

		d = &device{device: dev, name: dev[len(devicePrefix):]}
		if d.isPartition() {
			return nil, fmt.Errorf("device '%s' is a partition, it could not be initialized", d.device)
		}
		return d, nil
	} else {
		// something
		return d, nil
	}
}

// Puts drive to sleep/standby mode with ATA_OP_SLEEPNOW1, eventually ATA_OP_SLEEPNOW2.
func (d *device) putDriveToSleep() error {
	resp, err := sgioCommand(d.device, ATA_OP_SLEEPNOW1)
	if err != nil {
		if resp == EACCESS {
			return err
		}
		_, err = sgioCommand(d.device, ATA_OP_SLEEPNOW2)
	}
	return err
}

// Gets drive power mode/state with ATA_OP_CHECK_POWER_MODE1, eventually ATA_OP_CHECK_POWER_MODE2.
func (d *device) getDriveState() (powerMode, error) {
	val, err := sgioCommand(d.device, ATA_OP_CHECK_POWER_MODE1)
	if val == EACCESS {
		return unknown, err
	} else if val == byte(unknown) {
		val, err = sgioCommand(d.device, ATA_OP_CHECK_POWER_MODE2)
	}
	return powerMode(val), err
}

// Checks if drive power mode/state is standby with ATA_OP_CHECK_POWER_MODE1, eventually ATA_OP_CHECK_POWER_MODE2.
func (d *device) isDriveSleeping() (bool, error) {
	state, err := d.getDriveState()
	if err != nil {
		return false, err
	}
	return state == powerModeStandby, nil
}

// Updates major and minor from device specification. It handles possible hotswaping.
func (d *device) updateMajorMinor() error {
	fileInfo, err := os.Stat(d.device)
	if err != nil {
		d.reset()
		return fmt.Errorf("device '%s' is not available", d.device)
	}
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		d.reset()
		return fmt.Errorf("device '%s' is not available", d.device)
	}

	d.exists = true
	d.major = int(uint32(stat.Rdev >> 8))
	d.minor = int(uint32(stat.Rdev & 0xff))
	return nil
}

// Resets all values except device
func (d *device) reset() {
	d.exists = false
	d.major = 0
	d.minor = 0
	d.rIops = 0
	d.wIops = 0
	d.lastChange = 0
	d.lastStandBy = 0
	d.name = ""
	d.isSleeping = false
}

// Checks if defined device is partition
func (d *device) isPartition() bool {
	b, _ := os.Stat(path.Join("/sys/class/block/", d.name, "partition"))
	return b != nil
}

// Converts power mode into human readable text.
func (p *powerMode) stringify() string {
	switch *p {
	case powerModeStandby:
		return "standby"
	case powerModeNvcacheSpindown:
		return "NVcache_spindown"
	case powerModeNvcache_spinup:
		return "NVcache_spinup"
	case powerModeIdle:
		return "idle"
	case powerModeActive:
		return "active/idle"
	default:
		return "unknown"
	}
}
