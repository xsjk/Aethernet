package main

import (
	"Aethernet/pkg/iface"
	"fmt"
	"log"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func main() {

	var names = [2]string{"Aethernet", "本地连接* 2"}
	var filters = [2]string{"icmp and dst net 192.168.137.0/24", "icmp and dst net 172.18.1.0/24"}
	var handles [2]iface.Interface
	var mac_table = make(map[gopacket.Endpoint]gopacket.Endpoint)
	var err error

	handles[0], err = iface.OpenPCAP(names[0], filters[0])
	if err != nil {
		log.Fatalf("Unable to open %s: %v", names[0], err)
	}
	defer handles[0].Close()
	fmt.Printf("Listening on %s...\n", names[0])

	handles[1], err = iface.OpenPCAP(names[1], filters[1])
	if err != nil {
		log.Fatalf("Unable to open %s: %v", names[1], err)
	}
	defer handles[1].Close()
	fmt.Printf("Listening on %s...\n", names[1])

	go func() {
		for packet := range handles[0].Packets() {
			fmt.Printf("Packet from %s: %v\n", names[0], packet)
			ipv4 := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)

			if dstMAC, ok := mac_table[ipv4.NetworkFlow().Dst()]; !ok {
				fmt.Printf("MAC address not found for %v\n", ipv4.NetworkFlow().Dst())
			} else if srcMAC, ok := mac_table[ipv4.NetworkFlow().Src()]; !ok {
				fmt.Printf("MAC address not found for %v\n", ipv4.NetworkFlow().Src())
			} else {
				ethernetBuffer := gopacket.NewSerializeBuffer()
				gopacket.SerializeLayers(ethernetBuffer, gopacket.SerializeOptions{},
					&layers.Ethernet{
						SrcMAC:       net.HardwareAddr(srcMAC.Raw()),
						DstMAC:       net.HardwareAddr(dstMAC.Raw()),
						EthernetType: layers.EthernetTypeIPv4,
					},
					gopacket.Payload(packet.Data()),
				)
				handles[1].Write(ethernetBuffer.Bytes())
			}
		}
	}()

	go func() {
		for packet := range handles[1].Packets() {
			fmt.Printf("Packet from %s: %v\n", names[1], packet)
			ethernet := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			ipv4 := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)

			mac_table[ipv4.NetworkFlow().Dst()] = ethernet.LinkFlow().Dst()
			mac_table[ipv4.NetworkFlow().Src()] = ethernet.LinkFlow().Src()

			data := packet.Layer(layers.LayerTypeEthernet).LayerPayload()
			handles[0].Write(data)
		}
	}()

	fmt.Println("Press Enter to exit...")
	fmt.Scanln()

}
