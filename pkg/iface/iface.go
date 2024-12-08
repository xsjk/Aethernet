package iface

import "github.com/google/gopacket"

type Interface interface {
	Open() error
	Close()
	Packets() <-chan gopacket.Packet
	Write(data []byte) error
	Info() Info
}
