package iface

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/songgao/water"
)

type TAP struct {
	IP string

	iface *water.Interface
	info  Info

	frame   []byte
	packets chan gopacket.Packet
}

const FRAME_SIZE = 1600

func OpenTAP(ip string) (t *TAP, err error) {
	t = &TAP{IP: ip}
	return t, t.Open()
}

func (t *TAP) Open() (err error) {
	if t.iface, err = water.New(water.Config{DeviceType: water.TAP}); err != nil {
		return
	} else if t.info, err = GetInfo(t.iface.Name()); err != nil {
		return
	} else if err = t.info.SetIPv4(t.IP); err != nil {
		return
	}

	t.packets = make(chan gopacket.Packet)
	t.frame = make([]byte, FRAME_SIZE)
	go func() {
		var decoder gopacket.Decoder
		if t.iface.IsTAP() {
			decoder = layers.LayerTypeEthernet
		} else {
			decoder = layers.LayerTypeIPv4
		}
		for {
			if n, err := t.iface.Read(t.frame); err != nil {
				break
			} else {
				t.packets <- gopacket.NewPacket(t.frame[:n], decoder, gopacket.Default)
			}
		}
	}()
	return nil
}

func (t *TAP) Close() {
	t.iface.Close()
}

func (t *TAP) Packets() <-chan gopacket.Packet {
	return t.packets
}

func (t *TAP) Write(data []byte) error {
	_, err := t.iface.Write(data)
	return err
}

func (t *TAP) Info() Info {
	return t.info
}

func (t *TAP) LayerType() gopacket.LayerType {
	return layers.LayerTypeEthernet
}
