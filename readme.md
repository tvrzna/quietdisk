# QuietDisk
QuietDisk is a lightweight Go application designed to manage the power states of hard drives by monitoring their activity. It helps reduce power consumption and extend the lifespan of drives by transitioning them into standby mode when idle. The tool also provides features for checking the current power state of drives and manually putting them into sleep mode.

## Features
- Monitors hard drive activity and puts idle devices into standby mode.
- Directly interacts with devices using custom SG_IO and HDIO commands, avoiding external dependencies.
- List and check the power mode of devices without running the daemon.
- Option to manually put devices into standby mode.
- Set a timeout for transitioning devices to standby mode after inactivity.
- Prevents frequent toggling by enforcing a delay before a device can re-enter standby mode after waking up.
- Set a threshold for Input/Output Operations Per Second to determine when a device is idle.

## Usage
```
Usage: qd [options] [device ...]
Options:
	-h, --help			print this help
	-v, --version			print version
	-H, --hdd-only			works only with HDDs (rotational drives), skipping SSDs and NVMe devices.
	-l, --list			lists all available devices with their power mode
	-C, -c, --check			check power mode of listed devices
	-Y, --sleep			put listed devices into sleep mode
	-i, --idle [SECONDS]		sets idle period, before device is put into standby mode (default = 300)
	-g, --grace [SECONDS]		sets grace period, before device could be put into standby mode after return from standby mode (default = 600)
	-t, --treshold [IOPS]		sets IOPS treshold (default = 1)
	-V, --verbose			adds verbosity into logs
```

## Installation
Clone the repository and build the application using Go.

```bash
git clone https://github.com/tvrnza/quietdisk.git
cd quietdisk
make build install
```

## Example
Start monitoring /dev/sda with a 10-minute idle period and a grace period of 15 minutes:
```bash
qd -i 600 -g 900 /dev/sda
```

List available devices and their power states:
```bash
qd -l
```

Check the power state of a specific device (e.g., /dev/sda):
```bash
qd -C /dev/sda
```


Put a device into sleep mode manually:
```bash
qd -Y /dev/sda
```
