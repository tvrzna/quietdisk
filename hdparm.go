package main

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
)

type powerMode byte

const (
	standby powerMode = iota
	nvcache_spindown
	nvcache_spinup
	idle
	active
	unknown
)

func parsePowerMode(val string) powerMode {
	switch val {
	case "standby":
		return standby
	case "NVcache_spindown":
		return nvcache_spindown
	case "NVcache_spinup":
		return nvcache_spinup
	case "idle":
		return idle
	case "active/idle":
		return active
	default:
		return unknown
	}
}

func (p *powerMode) stringify() string {
	vals := []string{"standby", "NVcache_spindown", "NVcache_spinup", "idle", "active/idle", "unknown"}
	return vals[*p]
}

var pathHdparm string
var pathSudo string

func getExec(args ...string) (*exec.Cmd, error) {
	var err error

	if pathHdparm == "" {
		pathHdparm, err = exec.LookPath("hdparm")
		if err != nil {
			return nil, errors.New("hdparm is not installed")
		}
	}

	if os.Getegid() != 0 {
		if pathSudo == "" {
			pathSudo, err = exec.LookPath("sudo")
			if err != nil || pathSudo == "" {
				// Try doas, if sudo is not available
				pathSudo, err = exec.LookPath("doas")
				if err != nil {
					return nil, errors.New("sudo nor doas is available")
				}
			}
		}
	}

	command := pathHdparm
	if pathSudo != "" {
		command = pathSudo
		args = slices.Insert(args, 0, pathHdparm)
	}

	return exec.Command(command, args...), nil
}

func runHdparm(args ...string) (string, error) {
	cmd, err := getExec(args...)
	if err != nil {
		return "", err
	}

	var data []byte
	data, err = cmd.Output()

	return string(data), err

}

func putDriveToSleep(device string) error {
	_, err := runHdparm("-Y", device)
	return err
}

func getDriveState(device string) (powerMode, error) {
	output, err := runHdparm("-C", device)
	if err != nil {
		return unknown, err
	}
	i := strings.Index(output, "drive state is: ")
	if i < 0 {
		return unknown, errors.New("could not get drive state")
	}
	state := strings.TrimSpace(output[i+16:])
	return parsePowerMode(state), nil
}

func isDriveSleeping(device string) (bool, error) {
	state, err := getDriveState(device)
	if err != nil {
		return false, err
	}
	return state == standby, nil
}
