package iface

import (
	"fmt"
	"log"
	"net"
	"net/netip"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mdlayher/arp"
)

type Interface interface {
	Open() error
	Close()
	Packets() <-chan gopacket.Packet
	Write(data []byte) error
	Info() Info
	LayerType() gopacket.LayerType
}

var arpCache = make(map[netip.Addr]net.HardwareAddr)

func GetMAC(iface *net.Interface, ip net.IP) (mac net.HardwareAddr, err error) {

	ipaddr := netip.AddrFrom4([4]byte(ip.To4()))

	if mac, ok := arpCache[ipaddr]; ok {
		return mac, nil
	}

	client, err := arp.Dial(iface)
	if err != nil {
		return
	}
	defer client.Close()

	mac, err = client.Resolve(ipaddr)
	if err != nil {
		arpCache[ipaddr] = mac
	}

	return
}

func Send(iface Interface, packet gopacket.Packet) error {

	netiface, err := net.InterfaceByName(iface.Info().FriendlyName())
	if err != nil {
		panic(err)
	}

	buffer := gopacket.NewSerializeBuffer()
	switch iface.LayerType() {
	case layers.LayerTypeEthernet:

		layerEthernet, layerIPv4 := packet.Layer(layers.LayerTypeEthernet), packet.Layer(layers.LayerTypeIPv4)
		// if packet contains an Ethernet layer, send it directly
		if layerEthernet != nil {
			return iface.Write(packet.Data())

		} else if layerIPv4 != nil {

			fmt.Printf("Preparing MAC address for IPv4 packet\n")

			ipv4 := layerIPv4.(*layers.IPv4)
			// otherwise, create a new Ethernet layer
			srcMAC, err := GetMAC(netiface, ipv4.DstIP)
			if err != nil {
				return err
			}
			dstMAC, err := GetMAC(netiface, ipv4.SrcIP)
			if err != nil {
				return err
			}

			fmt.Printf("\t SrcMAC: %v\t DstMAC: %v\n", srcMAC, dstMAC)
			ethernet := &layers.Ethernet{
				SrcMAC:       srcMAC,
				DstMAC:       dstMAC,
				EthernetType: layers.EthernetTypeIPv4,
			}
			gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{},
				ethernet,
				ipv4,
				gopacket.Payload(ipv4.Payload),
			)

		} else {
			return fmt.Errorf("Unknown packet type")
		}

	case layers.LayerTypeIPv4:
		log.Printf("Sending IPv4 packet")
		if layer := packet.Layer(layers.LayerTypeIPv4); layer != nil {
			gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{},
				layer.(*layers.IPv4),
				gopacket.Payload(layer.LayerPayload()),
			)
		} else {
			return fmt.Errorf("Unknown packet type")
		}
	default:
		return fmt.Errorf("Unknown interface type")
	}

	return iface.Write(buffer.Bytes())
}
