package main

import (
	"Aethernet/cmd/project4/config"
	"Aethernet/pkg/async"
	"Aethernet/pkg/iface"
	"fmt"

	"github.com/google/gopacket/layers"
)

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
			packet, err := iface.DecodeIPPacket(data)
			if err != nil {
				fmt.Printf("Error decoding packet: %v\n", err)
				continue
			}
			icmpv4 := packet.Layer(layers.LayerTypeICMPv4)
			dns := packet.Layer(layers.LayerTypeDNS)
			tcp := packet.Layer(layers.LayerTypeTCP)
			if icmpv4 != nil || dns != nil || tcp != nil {
				fmt.Printf("Received packet from Aethernet: %v\n", packet)
				handle.Write(packet.Data())
			}
		}
	}()

	go func() {
		for packet := range handle.Packets() {
			icmpv4 := packet.Layer(layers.LayerTypeICMPv4)
			dns := packet.Layer(layers.LayerTypeDNS)
			tcp := packet.Layer(layers.LayerTypeTCP)
			if icmpv4 != nil || dns != nil || tcp != nil {
				fmt.Printf("Received packet from WinTUN: %v\n", packet)
				layer.Send(packet.Data())
			}
		}
	}()

	<-async.Exit()
	fmt.Println("Exiting...")

}
