package main

import (
	"Aethernet/cmd/project3/config"
	"Aethernet/pkg/async"
	"fmt"

	"github.com/google/gopacket/layers"
	"github.com/xsjk/go-wintun"
)

func main() {

	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Config: %+v\n", cfg)

	layer := config.CreateNaiveDataLinkLayer(cfg)
	iface := config.CreateWinTUN(cfg)

	layer.Open()
	defer layer.Close()

	err = iface.Open()
	if err != nil {
		fmt.Printf("Error opening TUN device: %v\n", err)
		return
	}
	defer iface.Close()

	go func() {
		for data := range layer.ReceiveAsync() {
			packet := wintun.Decode(data)
			icmpv4 := packet.Layer(layers.LayerTypeICMPv4)
			if icmpv4 != nil {
				fmt.Printf("Received packet from Aethernet: %v\n", packet)
				iface.Send(packet.Data())
			}
		}
	}()

	go func() {
		for data := range iface.ReceiveAsync() {
			packet := wintun.Decode(data)
			icmpv4 := packet.Layer(layers.LayerTypeICMPv4)
			if icmpv4 != nil {
				fmt.Printf("Received packet from WinTUN: %v\n", packet)
				layer.Send(packet.Data())
			}
		}
	}()

	<-async.EnterKey()
	fmt.Println("Exiting...")

}
