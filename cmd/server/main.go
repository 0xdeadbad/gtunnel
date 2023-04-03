package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/0xdeadbad/gtunnel/tuntap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	ifce, err := tuntap.NewIfBuilder().SetNoPktInfo().SetTap().Build()
	if err != nil {
		panic(err)
	}
	defer ifce.Close()

	err = ifce.SetIPv4([]byte{10, 0, 0, 1}, []byte{255, 255, 255, 0})
	if err != nil {
		panic(err)
	}

	err = ifce.Up()
	if err != nil {
		panic(err)
	}

	l, err := net.ListenPacket("udp", ":22122")
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		buf := make([]byte, 6)
		_, addr, err := l.ReadFrom(buf)
		if err != nil {
			log.Println(err)
			continue
		}
		if string(buf) != "hello" {
			continue
		}
		go _readLoop(ctx, cancel, ifce, l, addr)
		go _writeLoop(ctx, cancel, ifce, l, addr)
	}

}

func _readLoop(ctx context.Context, cancel context.CancelFunc, ifc tuntap.IfInterface, sv net.PacketConn, addr net.Addr) {
	buf := make([]byte, 1472)
	for {
		_, err := ifc.Read(buf)
		if err != nil {
			cancel()
			return
		}
		_, err = sv.WriteTo(buf, addr)
		if err != nil {
			cancel()
			return
		}
	}
}

func _writeLoop(ctx context.Context, cancel context.CancelFunc, ifc tuntap.IfInterface, sv net.PacketConn, addr net.Addr) {
	buf := make([]byte, 1472)
	for {
		_, _, err := sv.ReadFrom(buf)
		if err != nil {
			cancel()
			return
		}
		_, err = ifc.Write(buf)
		if err != nil {
			cancel()
			return
		}
	}
}
