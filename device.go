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

// Initializes device
func initDevice(dev string) (*device, error) {
	for dev[len(dev)-1] == '/' {
		dev = dev[:len(dev)-1]
	}

	d := &device{device: dev, name: dev[len(devicePrefix):]}
	if d.isPartition() {
		return nil, fmt.Errorf("device '%s' is a partition, it could not be initialized", d.device)
	}
	return d, nil
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
