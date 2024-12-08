package iface

import (
	"fmt"
	"log"

	"Aethernet/pkg/iface/kernel32"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/sys/windows"
	tun "golang.zx2c4.com/wintun"
)

type TUN struct {
	Name       string
	TunnelType string
	IP         string
	GUID       *windows.GUID

	adapter   *tun.Adapter
	session   tun.Session
	stopEvent windows.Handle
	readEvent windows.Handle
	channel   chan gopacket.Packet

	info Info
}

func OpenTUN(ip, name string) (t *TUN, err error) {
	t = &TUN{
		Name: name,
		IP:   ip,
		GUID: nil,
	}
	return t, t.Open()
}

func DecodeIPPacket(data []byte) (packet gopacket.Packet, err error) {
	var layerType gopacket.LayerType
	switch data[0] >> 4 {
	case 4:
		layerType = layers.LayerTypeIPv4
	case 6:
		layerType = layers.LayerTypeIPv6
	default:
		err = fmt.Errorf("Unknown IP version")
		return
	}
	packet = gopacket.NewPacket(data, layerType, gopacket.Lazy)
	return
}

func (t *TUN) Open() (err error) {

	t.adapter, err = tun.CreateAdapter(t.Name, t.TunnelType, t.GUID)
	if err != nil {
		return fmt.Errorf("Error creating adapter: %v", err)
	}
	defer func() {
		if err != nil {
			t.adapter.Close()
		}
	}()

	t.info, err = GetInfo(t.Name)
	if err != nil {
		return fmt.Errorf("Error getting adapter info for %s: %v", t.Name, err)
	}

	t.info.SetIPv4(t.IP)
	if err != nil {
		return fmt.Errorf("Error setting IP: %v", err)
	}

	t.session, err = t.adapter.StartSession(0x400000)
	if err != nil {
		return fmt.Errorf("Error starting session: %v", err)
	}

	t.stopEvent, _ = kernel32.CreateEvent(true, false, "StopEvent")
	t.readEvent = t.session.ReadWaitEvent()

	t.channel = make(chan gopacket.Packet)

	go func() {
		for {
			data, err := t.session.ReceivePacket()

			if err == nil {

				packet, err := DecodeIPPacket(data)
				if err == nil {
					t.channel <- packet
				} else {
					log.Printf("Error decoding packet: %v\n", err)
				}

			} else {
				switch err {
				case windows.ERROR_NO_MORE_ITEMS:
					res, err := kernel32.WaitForMultipleObjects([]windows.Handle{t.readEvent, t.stopEvent}, false, windows.INFINITE)
					switch res {
					case windows.WAIT_OBJECT_0:
						continue
					case windows.WAIT_OBJECT_0 + 1:
						return
					default:
						fmt.Printf("WaitForMultipleObjects failed: %v\n", err)
					}
				case windows.ERROR_HANDLE_EOF:
					fmt.Printf("%v, you should set the stopEvent before closing the session\n", err)
					return
				default:
					fmt.Printf("Unexpected error: %d %v\n", err, err)
					return
				}
			}
		}
	}()

	return

}

func (t *TUN) Close() {
	kernel32.SetEvent(t.stopEvent)
	defer kernel32.CloseHandle(t.stopEvent)
	t.session.End()
	t.adapter.Close()
}

func (t *TUN) Packets() <-chan gopacket.Packet {
	return t.channel
}

func (t *TUN) Write(data []byte) (err error) {
	buffer, err := t.session.AllocateSendPacket(len(data))
	if err == nil {
		copy(buffer, data)
		t.session.SendPacket(buffer)
	}
	return
}

func (t *TUN) Info() Info {
	return t.info
}

func (t *TUN) WaitForExit(duration uint32) bool {
	res, _ := kernel32.WaitForSingleObject(t.stopEvent, duration)
	switch res {
	case windows.WAIT_OBJECT_0:
		return true
	}
	return false
}

func (t *TUN) LayerType() gopacket.LayerType {
	return layers.LayerTypeIPv4
}
