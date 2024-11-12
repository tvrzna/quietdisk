package main

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
)

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

func isDriveSleeping(device string) (bool, error) {
	output, err := runHdparm("-C", device)
	if err != nil {
		return false, err
	}
	i := strings.Index(output, "drive state is: ")
	if i < 0 {
		return false, errors.New("could not get drive state")
	}
	state := strings.TrimSpace(output[i+16:])
	return state == "standby", nil
}
