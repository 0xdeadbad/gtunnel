package tuntap

import (
	"log"
	"net"
	"testing"
	"time"
)

// To run the test it's needed networking capabilities or run as su
func TestTunTap(t *testing.T) {
	ifc, err := NewIfBuilder().SetFlag(IF_TAP).SetFlag(IF_NO_PKT_INFO).SetName("").Build()
	if err != nil {
		t.Error(err)
	}

	err = ifc.SetIPv4(net.IPv4(10, 0, 0, 123), net.IPv4(255, 255, 255, 0))
	if err != nil {
		t.Error(err)
	}

	err = ifc.Up()
	if err != nil {
		t.Error(err)
	}

	t.Log("Up")

	err = ifc.SetMTU(1400)
	if err != nil {
		t.Error(err)
	}

	t.Log("MTU set to 1400")

	buffer := make([]byte, 1400)
	_, err = ifc.Read(buffer)
	if err != nil {
		t.Error(err)
	}

	<-time.After(10 * time.Second)

	hw, err := net.ParseMAC("18:c0:4d:64:51:7f")
	if err != nil {
		t.Error(err)
	}

	err = ifc.SetHWAddr(hw)
	if err != nil {
		t.Error(err)
	}

	<-time.After(5 * time.Second)

	err = ifc.Down()
	if err != nil {
		t.Error(err)
	}

	t.Log("Down")

	<-time.After(5 * time.Second)

	log.Println(buffer)

	err = ifc.Close()
	if err != nil {
		t.Error(err)
	}
}
