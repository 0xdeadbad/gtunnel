package tuntap

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

type iflags struct {
	name  [unix.IFNAMSIZ]byte
	flags uint16
}

type VirtIf struct {
	fd    int
	name  string
	flags uint16
	ifreq *unix.Ifreq
}

type IfBuilder struct {
	v     VirtIf
	flags IfFlag
	name  string
}

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

func (b IfBuilder) SetFlag(f IfFlag) IfBuilder {
	switch f {
	case IF_TAP:
		if IfFlag(b.flags&IF_TUN) == IF_TUN {
			b.flags ^= IF_TUN
		}
	case IF_TUN:
		if IfFlag(b.flags&IF_TAP) == IF_TAP {
			b.flags ^= IF_TAP
		}
	}
	b.flags |= f
	return b
}

func (b IfBuilder) SetName(name string) IfBuilder {
	b.name = name
	return b
}

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
	/*err = unix.IoctlSetInt(b.v.fd, unix.TUNSETPERSIST, 1)
	if err != nil {
		return nil, err
	}
	err = unix.IoctlSetInt(b.v.fd, unix.TUNSETOWNER, unix.Geteuid())
	if err != nil {
		return nil, err
	}
	err = unix.IoctlSetInt(b.v.fd, unix.TUNSETGROUP, unix.Getegid())
	if err != nil {
		return nil, err
	}*/
	b.v.ifreq = ifreq
	return &b.v, nil
}

func (v *VirtIf) Up() error {
	var ifl iflags
	s, e := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return e
	}
	defer unix.Close(s)
	copy(ifl.name[:], []byte(v.name))
	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return unix.Errno(ep)
	}
	ifl.flags |= uint16(IF_UP)
	_, _, ep = unix.Syscall(unix.SYS_IOCTL, uintptr(s), unix.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
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

type IfFlag uint16

const (
	IF_TUN         IfFlag = unix.IFF_TUN
	IF_TAP         IfFlag = unix.IFF_TAP
	IF_NO_PKT_INFO IfFlag = unix.IFF_NO_PI
	IF_MULTI_QUEUE IfFlag = unix.IFF_MULTI_QUEUE
	IF_UP          IfFlag = unix.IFF_UP
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
