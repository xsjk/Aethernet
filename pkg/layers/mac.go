package layer

import (
	"fmt"
	"time"
)

type MACAddress uint8

type MACType uint8

const (
	MACTypeData MACType = iota
	MACTypeACK
)

// Source (3 bit) | Destination (3 bit) | Type (1 bit) | IsLast (1 bit) | Index (8 bit)
type MACHeader struct {
	Source      MACAddress
	Destination MACAddress
	Type        MACType
	IsLast      bool
	Index       uint8
}

func (m MACHeader) Validate() {
	if m.Source&0x7 != m.Source {
		panic("Invalid source address")
	}
	if m.Destination&0x7 != m.Destination {
		panic("Invalid destination address")
	}
	if m.Type&0x1 != m.Type {
		panic("Invalid type")
	}
}

func (m MACHeader) ToBytes() []byte {
	m.Validate()
	bytes := []byte{byte(m.Source)<<5 | byte(m.Destination)<<2 | byte(m.Type)<<1, m.Index}
	if m.IsLast {
		bytes[0] |= 0x1
	}
	return bytes
}

func (m *MACHeader) FromBytes(data []byte) {
	m.Source = MACAddress(data[0] >> 5)
	m.Destination = MACAddress((data[0] >> 2) & 0x7)
	m.Type = MACType((data[0] >> 1) & 0x1)
	m.IsLast = (data[0] & 0x1) == 1
	m.Index = data[1]
	m.Validate()
}

func (m MACHeader) NumBytes() int {
	return 2
}

type MACLayer struct {
	PhysicalLayer

	Address    MACAddress
	ACKTimeout time.Duration

	// Send
	receivedACK   chan struct{}
	expectedIndex uint8

	// Receive
	currentPacket []byte
	OutputChan    chan []byte
}

func (m *MACLayer) Open() {
	m.PhysicalLayer.Open()
	go func() {
		for {
			packet := m.PhysicalLayer.Receive()
			header := MACHeader{}
			header.FromBytes(packet[:header.NumBytes()])
			if header.Destination == m.Address {
				m.Handle(header, packet[header.NumBytes():])
			}
		}
	}()
}

func (m *MACLayer) Handle(header MACHeader, data []byte) {

	// // ignore the packet if the packet is sent by myself
	// if header.Source == m.Address {
	// 	return
	// }

	switch header.Type {
	case MACTypeData:
		m.currentPacket = append(m.currentPacket, data...)
		if header.IsLast {
			select {
			case m.OutputChan <- m.currentPacket:
				fmt.Println("[MAC] Packet received")
			default:
				fmt.Println("[MAC] Output channel is full")
				panic("Output channel is full")
			}
			m.currentPacket = nil
		}
		// send the ACK
		go m.PhysicalLayer.Send(MACHeader{
			Source:      m.Address,
			Destination: header.Source,
			Type:        MACTypeACK,
			Index:       header.Index,
		}.ToBytes())
	case MACTypeACK:
		// check the index with the current sending packet
		fmt.Printf("[MAC] ACK received for packet %d\n", header.Index)
		if m.expectedIndex == header.Index {
			close(m.receivedACK)
		} else {
			fmt.Printf("[MAC] ACK for packet %d is not expected\n", header.Index)
			panic("ACK for packet is not expected")
		}
	}
}

func (m *MACLayer) Send(address MACAddress, data []byte) error {

	packetLength := m.PhysicalLayer.Encoder.Modulator.BytePerFrame

	// split the data into packets (do not use physical layer's packet splitting)
	packets := make([][]byte, 0)
	header := MACHeader{
		Source:      m.Address,
		Destination: address,
		Type:        MACTypeData,
	}
	for i := 0; i < len(data); i += packetLength {
		end := min(i+packetLength, len(data))
		if end == len(data) {
			header.IsLast = true
		}
		packet := append(header.ToBytes(), data[i:end]...)
		packets = append(packets, packet)
		header.Index++
	}

	// send the packets
	for i, packet := range packets {
	resendLoop:
		for {
			m.PhysicalLayer.Send(packet)
			m.receivedACK = make(chan struct{})
			m.expectedIndex = uint8(i)
			// wait for the ACK
			select {
			case <-m.receivedACK:
				// ACK received
				break resendLoop
			case <-time.After(m.ACKTimeout):
				// ACK timeout
				fmt.Printf("[MAC] Packet %d ACK timeout\n", i)
				continue
			}
		}
		fmt.Printf("[MAC] Packet %d sent and ACK received\n", i)
	}

	m.receivedACK = nil

	return nil

}

func (m *MACLayer) Receive() []byte {
	return <-m.OutputChan
}
