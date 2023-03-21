package tuntap

import (
	"log"
	"net"
	"testing"
	"time"
)

func TestTunTap(t *testing.T) {
	ifc, err := NewIfBuilder().SetFlag(IF_TAP).SetFlag(IF_NO_PKT_INFO).SetName("").Build()
	if err != nil {
		t.Error(err)
	}

	err = ifc.SetIPv4(net.IPv4(10, 0, 0, 123), net.IPv4(255, 255, 255, 0))
	if err != nil {
		t.Fatal(err)
	}

	err = ifc.Up()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Up")

	buffer := make([]byte, 1500)
	_, err = ifc.Read(buffer)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(10 * time.Second)

	err = ifc.Down()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Down")

	<-time.After(10 * time.Second)

	log.Println(buffer)
}
