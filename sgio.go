package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

type ataOp byte

const (
	SG_IO          = 0x2285
	HDIO_DRIVE_CMD = 0x031f

	SG_DXFER_NONE = 0

	ATA_16             byte = 0x85
	ATA_OP_DSM         byte = 0x06
	ATA_OP_READ_PIO    byte = 0x20
	ATA_OP_READ_VERIFY byte = 0x40

	ATA_OP_CHECK_POWER_MODE1 ataOp = 0xe5
	ATA_OP_SLEEPNOW1         ataOp = 0xe6
	ATA_OP_CHECK_POWER_MODE2 ataOp = 0x98
	ATA_OP_SLEEPNOW2         ataOp = 0x99
)

type HDDriveCmdHdr struct {
	Command  byte
	Feature  byte
	Nsect    byte
	LCyl     byte
	HCyl     byte
	Select   byte
	Control  byte
	Reserved [4]byte
}

type SgIoHdr struct {
	InterfaceID    int32
	DxferDirection int32
	CmdLen         uint8
	MxSbLen        uint8
	IovecCount     uint16
	DxferLen       uint32
	Dxferp         uintptr
	Cmdp           uintptr
	Sbp            uintptr
	Timeout        uint32
	Flags          uint32
	PackID         int32
	UsrPtr         uintptr
	Status         uint8
	MaskedStatus   uint8
	MsgStatus      uint8
	SbLenWr        uint8
	HostStatus     uint16
	DriverStatus   uint16
	Resid          int32
	Duration       uint32
	Info           uint32
}

func sgioCommand(device string, ataCommand ataOp) (byte, error) {
	file, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		log.Print(err)
		return byte(syscall.EACCES), err
	}
	defer file.Close()

	senseBuf := make([]byte, 32)
	cdb := make([]byte, 16)
	cdb[0] = ATA_16
	cdb[1] = ATA_OP_DSM
	cdb[2] = ATA_OP_READ_PIO
	cdb[13] = ATA_OP_READ_VERIFY
	cdb[14] = byte(ataCommand)

	ioHdr := &SgIoHdr{
		InterfaceID:    int32('S'),
		DxferDirection: SG_DXFER_NONE,
		CmdLen:         uint8(len(cdb)),
		MxSbLen:        uint8(len(senseBuf)),
		DxferLen:       0,
		Dxferp:         0,
		Cmdp:           uintptr(unsafe.Pointer(&cdb[0])),
		Sbp:            uintptr(unsafe.Pointer(&senseBuf[0])),
		Timeout:        5000,
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(SG_IO), uintptr(unsafe.Pointer(ioHdr)))
	if errno != 0 {
		if errno.Is(syscall.EINVAL) || errno.Is(syscall.ENODEV) || errno.Is(syscall.EBADE) {
			return hdioCommand(file.Fd(), ataCommand)
		}
		return byte(unknown), fmt.Errorf("ioctl SG_IO failed: %v", errno)
	}
	if senseBuf[0] != 0x72 {
		return byte(unknown), fmt.Errorf("SG_IO returned: 0x%x", senseBuf[0])
	}

	return senseBuf[13], nil
}

func hdioCommand(fd uintptr, ataCommand ataOp) (byte, error) {
	cmdHdr := &HDDriveCmdHdr{Command: byte(ataCommand)}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(HDIO_DRIVE_CMD), uintptr(unsafe.Pointer(cmdHdr)))

	if errno != 0 {
		return byte(unknown), fmt.Errorf("ioctl failed: %v", errno)
	}

	return cmdHdr.Nsect, nil
}
