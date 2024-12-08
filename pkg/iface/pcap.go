package iface

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type PCAP struct {
	Name   string
	Filter string

	handle *pcap.Handle
	info   Info
}

func OpenPCAP(name, filter string) (p *PCAP, err error) {
	p = &PCAP{
		Name:   name,
		Filter: filter,
	}
	err = p.Open()
	return
}

func (p *PCAP) Open() (err error) {
	p.info, err = GetInfo(p.Name)
	if err != nil {
		return
	}

	p.handle, err = pcap.OpenLive(p.info.PcapName(), 1600, true, pcap.BlockForever)
	if err != nil {
		return
	}

	err = p.handle.SetBPFFilter(p.Filter)
	return
}

func (p *PCAP) Close() {
	p.handle.Close()
}

func (p *PCAP) Packets() <-chan gopacket.Packet {
	return gopacket.NewPacketSource(p.handle, p.handle.LinkType()).Packets()
}

func (p *PCAP) Write(packet []byte) error {
	return p.handle.WritePacketData(packet)
}

func (p *PCAP) Info() Info {
	return p.info
}

func (p *PCAP) LayerType() gopacket.LayerType {
	switch p.handle.LinkType() {
	case layers.LinkTypeEthernet:
		return layers.LayerTypeEthernet
	case layers.LinkTypeIPv4, layers.LinkTypeRaw, 12:
		return layers.LayerTypeIPv4
	default:
		return gopacket.LayerTypeZero
	}
}
