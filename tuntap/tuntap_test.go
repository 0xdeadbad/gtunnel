package tuntap

import (
	"log"
	"testing"
	"time"
)

func TestTunTap(t *testing.T) {
	ifc, err := NewIfBuilder().SetFlag(IF_TAP).SetFlag(IF_NO_PKT_INFO).SetName("").Build()
	if err != nil {
		t.Error(err)
	}
	err = ifc.Up()
	if err != nil {
		t.Fatal(err)
	}

	buffer := make([]byte, 1500)
	_, err = ifc.Read(buffer)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(10 * time.Second)

	log.Println(buffer)
}
