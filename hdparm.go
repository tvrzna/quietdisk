package main

import (
	"errors"
	"os"
	"os/exec"
	"slices"
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

func runHdparm(args ...string) error {
	cmd, err := getExec(args...)
	if err != nil {
		return err
	}

	return cmd.Run()
}

func putDriveToSleep(device string) error {
	return runHdparm("-Y", device)
}
