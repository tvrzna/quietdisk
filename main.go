package main

import "os"

func main() {
	c := initContext(os.Args)

	switch c.action {
	case contextActionList:
		c.printListedDevices()
	case contextActionCheck:
		c.checkDevices()
	case contextActionSleep:
		c.sleepDevices()
	default:
		c.startDaemon()
	}
}
