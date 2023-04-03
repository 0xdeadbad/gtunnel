package tuntap

import (
	"io"
	"net"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	nameSize     = unix.IFNAMSIZ
	sockaddrSize = unsafe.Sizeof(unix.RawSockaddrInet4{})
	ipSize       = 4
)

type ifreq [40]byte

func (f *ifreq) SetName(name string) int {
	return copy(f[:nameSize], name)
}

func (f *ifreq) SetSockaddr(family, port uint16, addr net.IP) error {
	fam := uint16ToBytes(family)
	p := uint16ToBytes(port)

	copy(f[nameSize:], fam[:])
	copy(f[nameSize+2:], p[:])
	copy(f[nameSize+4:], addr.To4())
	copy(f[nameSize+8:], []uint8{0, 0, 0, 0, 0, 0, 0, 0})

	return nil
}

func (f *ifreq) SetHWAddr(mac net.HardwareAddr) error {
	fam := uint16ToBytes(unix.ARPHRD_ETHER)

	copy(f[nameSize:], fam[:])
	copy(f[nameSize+2:], mac[:])

	return nil
}

func (f *ifreq) SetFlag(flag IfFlag) {
	fl := uint16ToBytes(uint16(flag))

	f[nameSize] |= fl[0]
	f[nameSize+1] |= fl[1]
}

func (f *ifreq) UnsetFlag(flag IfFlag) {
	fl := uint16ToBytes(uint16(flag))

	f[nameSize] &= ^fl[0]
	f[nameSize+1] &= ^fl[1]
}

func (f *ifreq) SetMTU(mtu int32) {
	m := uint32ToBytes(mtu)
	copy(f[nameSize:], m[:])
}

func uint16ToBytes(n uint16) [2]byte {
	return [2]byte{byte(n & 0b0000000011111111), byte(n >> 8)}
}

func uint32ToBytes(n int32) [4]byte {
	return [4]byte{byte(n & 0b00000000000000000000000011111111), byte((n & 0b00000000000000001111111100000000) >> 8), byte((n & 0b00000000111111110000000000000000) >> 16), byte(n >> 24)}
}

/*func bytesToUint16(n [2]byte) uint16 {
	return (uint16(n[1]) << 8) | uint16(n[0])
}*/

// Struct object that represents the TUN/TAP Virtual Interface
type VirtIf struct {
	fd    int
	name  string
	flags uint16
	ifreq *unix.Ifreq
	io.ReadWriteCloser
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
func (b IfBuilder) Build() (IfInterface, error) {
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
	var ifl ifreq
	s, e := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return e
	}
	defer unix.Close(s)
	ifl.SetName(v.name)
	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	ifl.SetFlag(IFF_UP)
	_, _, ep = unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	return nil
}

// Set Virtual Interface Down
func (v *VirtIf) Down() error {
	var ifl ifreq
	s, e := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return e
	}
	defer unix.Close(s)
	ifl.SetName(v.name)
	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	ifl.UnsetFlag(IFF_UP)
	_, _, ep = unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	return nil
}

// Set Virtual Interface IPv4 address
func (v *VirtIf) SetIPv4(ip net.IP, mask net.IP) error {
	var addr_in ifreq
	s, e := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return e
	}
	defer unix.Close(s)
	addr_in.SetName(v.name)
	addr_in.SetSockaddr(unix.AF_INET, 0, ip)
	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFADDR, uintptr(unsafe.Pointer(&addr_in)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	addr_in.SetSockaddr(unix.AF_INET, 0, mask)
	_, _, ep = unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFNETMASK, uintptr(unsafe.Pointer(&addr_in)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	return nil
}

func (v *VirtIf) SetMTU(mtu int32) error {
	var ifl ifreq
	s, e := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return e
	}
	defer unix.Close(s)
	ifl.SetName(v.name)
	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCGIFMTU, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	ifl.SetMTU(mtu)
	_, _, ep = unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFMTU, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	return nil
}

// Set Virtual Interface MAC address
func (v *VirtIf) SetHWAddr(mac net.HardwareAddr) error {
	var hw_addr ifreq
	s, e := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return e
	}
	defer unix.Close(s)
	hw_addr.SetName(v.name)
	hw_addr.SetHWAddr(mac)
	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFHWADDR, uintptr(unsafe.Pointer(&hw_addr)))
	if ep != 0 {
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

type IfInterface interface {
	io.ReadWriteCloser
	SetIPv4(net.IP, net.IP) error
	SetMTU(mtu int32) error
	SetHWAddr(mac net.HardwareAddr) error
	Up() error
	Down() error
}
