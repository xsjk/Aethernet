package main

import (
	"Aethernet/cmd/project4/config"
	"Aethernet/pkg/async"
	"Aethernet/pkg/iface"
	"fmt"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var allowedDNSQueries = []string{"baidu", "example"}

func allow(packet gopacket.Packet) (allow bool) {

	icmpv4 := packet.Layer(layers.LayerTypeICMPv4)
	dns := packet.Layer(layers.LayerTypeDNS)
	tcp := packet.Layer(layers.LayerTypeTCP)
	if icmpv4 != nil {
		allow = true
	} else if dns != nil {
		dnsLayer := dns.(*layers.DNS)
		for _, query := range dnsLayer.Questions {
			if query.Type != layers.DNSTypeA {
				return false
			}
			for _, allowQuery := range allowedDNSQueries {
				if strings.Contains(string(query.Name), allowQuery) {
					allow = true
					break
				}
			}
		}
	} else if tcp != nil {
		allow = true
	}
	return
}

func main() {

	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Config: %+v\n", cfg)

	layer := config.CreateNaiveDataLinkLayer(cfg)
	handle, err := config.OpenInterface(cfg)
	if err != nil {
		fmt.Printf("Error opening interface: %v\n", err)
		return
	}

	layer.Open()
	defer layer.Close()

	err = handle.Open()
	if err != nil {
		fmt.Printf("Error opening TUN device: %v\n", err)
		return
	}
	defer handle.Close()

	go func() {
		for data := range layer.ReceiveAsync() {
			packet, _ := iface.DecodeIPPacket(data)
			if packet != nil && allow(packet) {
				fmt.Printf("Received packet from Aethernet: %v\n", packet)
				handle.Write(packet.Data())
			}
		}
	}()

	go func() {
		for packet := range handle.Packets() {
			if allow(packet) {
				fmt.Printf("Received packet from WinTUN: %v\n", packet)
				layer.Send(packet.Data())
			}
		}
	}()

	<-async.Exit()
	fmt.Println("Exiting...")

}
