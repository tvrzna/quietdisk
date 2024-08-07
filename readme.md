# QuietDisk
QuietDisk is a simple Go application designed to monitor the Read and Write IOPS (Input/Output Operations Per Second) of a specified hard drive every minute. When the drive is detected to be idle for a specified period, QuietDisk will automatically put the drive into standby mode using `hdparm -Y`. This helps in reducing power consumption and prolonging the lifespan of your hard drives.

## Features:
- Monitors Read and Write IOPS of the specified hard drive
- Automatically transitions idle drives into standby mode
- Configurable idle period and IOPS threshold
- Implements a grace period to avoid frequent state toggling
- Lightweight and efficient, suitable for continuous monitoring

## Usage:
```
Usage: qd [options] [device ...]
Options:
	-h, --help			print this help
	-v, --version			print version
	-i, --idle [SECONDS]		sets idle period, before device is put into standby mode (default = 300)
	-g, --grace [SECONDS]		sets grace period, before device could be put into standby mode after return from standby mode (default = 600)
	-t, --treshold [IOPS]		sets IOPS treshold (default = 1)
	-V, --verbose			adds verbosity into logs
```

## Installation:
Clone the repository and build the application using Go.

```bash
git clone https://github.com/tvrnza/quietdisk.git
cd quietdisk
make build install
```