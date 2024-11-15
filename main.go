package main

import "os"

func main() {
	c := initContext(os.Args)
	c.startDaemon()
}
