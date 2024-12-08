package main

import (
	"Aethernet/pkg/async"
	"Aethernet/pkg/iface"
	"fmt"
	"log"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type NATKey struct {
	IP gopacket.Endpoint
	ID uint16
}

type NATTable struct {
	table     map[NATKey]uint16
	inv_table map[uint16]NATKey
}

func MakeNATTable() NATTable {
	return NATTable{
		table:     make(map[NATKey]uint16),
		inv_table: make(map[uint16]NATKey),
	}
}

func (n *NATTable) AllocateOutbound(ip net.IP, id uint16) uint16 {
	endpoint := gopacket.NewEndpoint(layers.EndpointIPv4, ip)
	key := NATKey{IP: endpoint, ID: id}
	var out_port uint16
	for out_port = 1024; out_port <= 65535; out_port++ {
		if _, ok := n.inv_table[out_port]; !ok {
			n.table[key] = out_port
			n.inv_table[out_port] = key
			return out_port
		}
	}
	panic("No available outbound port")
}

func (n *NATTable) DeallocateOutbound(port uint16) {
	in_port, ok := n.inv_table[port]
	if !ok {
		panic("Port not found in NAT table")
	}
	delete(n.table, in_port)
	delete(n.inv_table, port)
}

func (n *NATTable) GetOutbound(ip net.IP, id uint16) (uint16, bool) {
	endpoint := gopacket.NewEndpoint(layers.EndpointIPv4, ip)
	key := NATKey{IP: endpoint, ID: id}
	out_port, ok := n.table[key]
	return out_port, ok
}

func (n *NATTable) GetInbound(port uint16) (net.IP, uint16, bool) {
	key, ok := n.inv_table[port]
	return net.IP(key.IP.Raw()), key.ID, ok
}

func (n *NATTable) OccupyOutbound(port uint16) {
	_, ok := n.inv_table[port]
	if ok {
		panic("Port already occupied")
	}
	n.inv_table[port] = NATKey{}
}

func main() {

	var names = [2]string{"Aethernet", "WLAN"}
	var filters = [2]string{"icmp and ip dst 1.1.1.1", "icmp and ip src 1.1.1.1"}
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

	nat_table := MakeNATTable()

	go func() {
		for packet := range handles[0].Packets() {
			fmt.Printf("Packet from %s: %v\n", names[0], packet)
			ipv4 := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
			icmp := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)

			switch icmp.TypeCode.Type() {
			case layers.ICMPv4TypeEchoRequest:
				outbound, ok := nat_table.GetOutbound(ipv4.SrcIP, icmp.Id)
				if !ok {
					log.Printf("Allocating new outbound port for ICMP ID %d\n", icmp.Id)
					outbound = nat_table.AllocateOutbound(ipv4.SrcIP, icmp.Id)
				}

				buffer := gopacket.NewSerializeBuffer()

				// make NAT packet
				dstIP := ipv4.DstIP
				srcIP, _ := handles[1].Info().GetIPv4()

				srcMac, ok := mac_table[gopacket.NewEndpoint(layers.EndpointIPv4, srcIP)]
				if !ok {
					log.Printf("MAC address not found for %v\n", srcIP)
					continue
				}
				dstMac, ok := mac_table[gopacket.NewEndpoint(layers.EndpointIPv4, dstIP)]
				if !ok {
					log.Printf("MAC address not found for %v\n", dstIP)
					continue
				}

				ipv4.SrcIP = srcIP
				icmp.Id = outbound

				gopacket.SerializeLayers(
					buffer,
					gopacket.SerializeOptions{
						FixLengths:       false,
						ComputeChecksums: true,
					},
					&layers.Ethernet{
						SrcMAC:       srcMac.Raw(),
						DstMAC:       dstMac.Raw(),
						EthernetType: layers.EthernetTypeIPv4,
					},
					ipv4,
					icmp,
					gopacket.Payload(icmp.Payload),
				)

				handles[1].Write(buffer.Bytes())
			}

		}
	}()

	go func() {
		for packet := range handles[1].Packets() {
			fmt.Printf("Packet from %s: %v\n", names[1], packet)
			ethernet := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			ipv4 := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
			icmp := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)

			my_ip, _ := handles[1].Info().GetIPv4()
			mac_table[gopacket.NewEndpoint(layers.EndpointIPv4, my_ip)] = ethernet.LinkFlow().Dst()
			mac_table[ipv4.NetworkFlow().Src()] = ethernet.LinkFlow().Src()

			switch icmp.TypeCode.Type() {
			case layers.ICMPv4TypeEchoReply:
				inboundIP, icmpId, ok := nat_table.GetInbound(icmp.Id)
				if !ok {
					log.Printf("No inbound mapping found for ICMP ID %d, not the reply to a NATed packet\n", icmp.Id)
					nat_table.OccupyOutbound(icmp.Id)
					continue
				}

				buffer := gopacket.NewSerializeBuffer()

				// NAT
				ipv4.DstIP = inboundIP
				icmp.Id = icmpId

				gopacket.SerializeLayers(
					buffer,
					gopacket.SerializeOptions{
						FixLengths:       false,
						ComputeChecksums: true,
					},
					ipv4,
					icmp,
					gopacket.Payload(icmp.Payload),
				)

				handles[0].Write(buffer.Bytes())
			}
		}
	}()

	<-async.Exit()

}
