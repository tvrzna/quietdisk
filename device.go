package main

import (
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
}

// Initializes device
func initDevice(dev string) (*device, error) {
	d := &device{device: dev}
	return d, nil
}

// Updates major and minor from device specification. It handles possible hotswaping.
func (d *device) updateMajorMinor() {
	fileInfo, err := os.Stat(d.device)
	if err != nil {
		d.reset()
		return
	}
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		d.reset()
		return
	}

	d.exists = true
	d.major = int(uint32(stat.Rdev >> 8))
	d.minor = int(uint32(stat.Rdev & 0xff))
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
}

// Checks if defined device is partition
func (d *device) isPartition() bool {
	b, _ := os.Stat(path.Join("/sys/class/block/", d.name, "partition"))
	return b != nil
}
