package tuntap

import (
	"fmt"
	"net"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

type iflags struct {
	name  [unix.IFNAMSIZ]byte
	flags uint16
}

type iflagsaddr struct {
	name [unix.IFNAMSIZ]byte
	unix.RawSockaddrInet4
}

// Struct object that represents the TUN/TAP Virtual Interface
type VirtIf struct {
	fd    int
	name  string
	flags uint16
	ifreq *unix.Ifreq
}

// VirtIf builder. Can chain options and settings to configure the new VirtIf
type IfBuilder struct {
	v     VirtIf
	flags IfFlag
	name  string
}

// Creates a new VirtIf builder without any parameters preset
func NewIfBuilder() IfBuilder {
	return IfBuilder{
		flags: 0,
		name:  "",
		v: VirtIf{
			fd:   -1,
			name: "",
		},
	}
}

// Set Virtual Interface as TAP, removing the TUN flag if present
func (b IfBuilder) SetTap() IfBuilder {
	if IfFlag(b.flags&IF_TUN) == IF_TUN {
		b.flags &= ^IF_TUN
	}
	b.flags |= IF_TAP

	return b
}

// Set Virtual Interface as TUN, removing the TAP flag if present
func (b IfBuilder) SetTun() IfBuilder {
	if IfFlag(b.flags&IF_TAP) == IF_TAP {
		b.flags &= ^IF_TAP
	}
	b.flags |= IF_TUN

	return b
}

// The the Virtual Interface to not contain packets info
func (b IfBuilder) SetNoPktInfo() IfBuilder {
	b.flags |= IF_NO_PKT_INFO

	return b
}

// Set manually a flag
func (b IfBuilder) SetFlag(f IfFlag) IfBuilder {
	b.flags |= f
	return b
}

// Set the Virtual Interface name
func (b IfBuilder) SetName(name string) IfBuilder {
	b.name = name
	return b
}

// Build returns the Virtual Interface with the specified flags and options.
func (b IfBuilder) Build() (*VirtIf, error) {
	fd, err := unix.Open("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	b.v.fd = fd
	b.v.flags = uint16(b.flags)
	ifreq, err := unix.NewIfreq(b.name)
	if err != nil {
		return nil, err
	}
	ifreq.SetUint16(uint16(b.flags))
	err = unix.IoctlIfreq(b.v.fd, unix.TUNSETIFF, ifreq)
	if err != nil {
		return nil, err
	}
	b.v.name = ifreq.Name()
	err = unix.IoctlSetInt(b.v.fd, unix.TUNSETOWNER, unix.Geteuid())
	if err != nil {
		return nil, err
	}
	err = unix.IoctlSetInt(b.v.fd, unix.TUNSETGROUP, unix.Getegid())
	if err != nil {
		return nil, err
	}
	/*err = unix.IoctlSetInt(b.v.fd, unix.TUNSETPERSIST, 1)
	if err != nil {
		return nil, err
	}
	*/
	b.v.ifreq = ifreq
	return &b.v, nil
}

// Set Virtual Interface Up
func (v *VirtIf) Up() error {
	var ifl iflags
	s, e := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return e
	}
	defer unix.Close(s)
	copy(ifl.name[:], v.name)
	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	ifl.flags |= uint16(IFF_UP)
	_, _, ep = unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	return nil
}

// Set Virtual Interface Down
func (v *VirtIf) Down() error {
	var ifl iflags
	s, e := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return e
	}
	defer unix.Close(s)
	copy(ifl.name[:], v.name)
	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	ifl.flags &= ^uint16(IFF_UP)
	_, _, ep = unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	return nil
}

// Set Virtual Interface IPv4 address
func (v *VirtIf) SetIPv4(ip net.IP) error {
	var addr_in iflagsaddr
	s, e := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		fmt.Println("1")
		return e
	}
	defer unix.Close(s)
	copy(addr_in.name[:], v.name)
	addr_in.Family = unix.AF_INET
	copy(addr_in.Addr[:], ip.To4())
	addr_in.Port = 0
	copy(addr_in.Zero[:], []uint8{0, 0, 0, 0, 0, 0, 0, 0})
	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFADDR, uintptr(unsafe.Pointer(&addr_in)))
	if ep != 0 {
		fmt.Println("3")
		return unix.Errno(ep)
	}
	return nil
}

func (v *VirtIf) Write(b []byte) (int, error) {
	return unix.Write(v.fd, b)
}

func (v *VirtIf) Read(b []byte) (int, error) {
	return unix.Read(v.fd, b)
}

func (v *VirtIf) Close() error {
	return unix.Close(v.fd)
}

// Type for setting the VirtIf flags
type IfFlag uint16

const (
	IF_TUN         IfFlag = unix.IFF_TUN
	IF_TAP         IfFlag = unix.IFF_TAP
	IF_NO_PKT_INFO IfFlag = unix.IFF_NO_PI
	//	IF_MULTI_QUEUE IfFlag = unix.IFF_MULTI_QUEUE
)

// Type for setting the If flags itself
type IffFlag uint16

const (
	IFF_UP      = unix.IFF_UP
	IFF_RUNNING = unix.IFF_RUNNING
)

type ConfingInvalid string

const (
	INVALID_FLAG_COMBINATION ConfingInvalid = "invalid flags combination"
)

func (c ConfingInvalid) toString() string {
	return string(c)
}

func (c ConfingInvalid) Error() string {
	return c.toString()
}
