package main

import "syscall"

type powerMode byte

const (
	standby          powerMode = 0x00
	nvcache_spindown powerMode = 0x40
	nvcache_spinup   powerMode = 0x41
	idle             powerMode = 0x80
	active           powerMode = 0xff
	unknown          powerMode = 0xf0
)

func (p *powerMode) stringify() string {
	switch *p {
	case standby:
		return "standby"
	case nvcache_spindown:
		return "NVcache_spindown"
	case nvcache_spinup:
		return "NVcache_spinup"
	case idle:
		return "idle"
	case active:
		return "active/idle"
	default:
		return "unknown"
	}
}

func putDriveToSleep(device string) error {
	resp, err := sgioCommand(device, ATA_OP_SLEEPNOW1)
	if err != nil {
		if resp == byte(syscall.EACCES) {
			return err
		}
		_, err = sgioCommand(device, ATA_OP_SLEEPNOW2)
	}
	return err
}

func getDriveState(device string) (powerMode, error) {
	val, err := sgioCommand(device, ATA_OP_CHECK_POWER_MODE1)
	if val == byte(syscall.EACCES) {
		return unknown, err
	} else if val == byte(unknown) {
		val, err = sgioCommand(device, ATA_OP_CHECK_POWER_MODE2)
	}
	return powerMode(val), err
}

func isDriveSleeping(device string) (bool, error) {
	state, err := getDriveState(device)
	if err != nil {
		return false, err
	}
	return state == standby, nil
}
