package layers

import (
	"Aethernet/pkg/async"
	"fmt"
	"time"

	"golang.org/x/exp/rand"
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

type BackoffTimer interface {
	GetBackoffTime(retries int) time.Duration
}

type RandomBackoffTimer struct {
	MinDelay time.Duration
	MaxDelay time.Duration
}

func (b RandomBackoffTimer) GetBackoffTime(retries int) time.Duration {
	return b.MinDelay + time.Duration(rand.Int63n(int64(b.MaxDelay-b.MinDelay)))
}

type MACLayer struct {
	PhysicalLayer

	BytePerFrame int

	Address      MACAddress
	ACKTimeout   time.Duration
	MaxRetries   int
	BackoffTimer BackoffTimer

	// Send
	receivedACK   chan struct{}
	expectedIndex uint8

	// Receive
	currentPacket []byte
	OutputChan    chan []byte
}

func (m *MACLayer) Open() {
	m.PhysicalLayer.Open()
	m.receivedACK = make(chan struct{})
	go func() {
		for {
			packet := m.PhysicalLayer.Receive()
			header := MACHeader{}
			header.FromBytes(packet[:header.NumBytes()])
			if header.Destination == m.Address {
				m.handle(header, packet[header.NumBytes():])
			}
		}
	}()
}

func (m *MACLayer) handle(header MACHeader, data []byte) {

	switch header.Type {
	case MACTypeData:
		m.currentPacket = append(m.currentPacket, data...)
		if header.IsLast {
			select {
			case m.OutputChan <- m.currentPacket:
				fmt.Printf("[MAC%x] Packet %d received\n", m.Address, header.Index)
			default:
				fmt.Printf("[MAC%x] Output channel is full\n", m.Address)
				panic("Output channel is full")
			}
			m.currentPacket = nil
		}
		// send the ACK
		fmt.Printf("[MAC%x] ACK for packet %d sent\n", m.Address, header.Index)
		go m.PhysicalLayer.Send(MACHeader{
			Source:      m.Address,
			Destination: header.Source,
			Type:        MACTypeACK,
			Index:       header.Index,
		}.ToBytes())
	case MACTypeACK:
		// check the index with the current sending packet
		fmt.Printf("[MAC%x] ACK for packet %d received\n", m.Address, header.Index)
		if m.expectedIndex == header.Index {

			// make sure m.receiveACK is not closed
			select {
			case <-m.receivedACK:
				panic("m.receivedACK is not open")
			default:
			}

			select {
			case m.receivedACK <- struct{}{}:
				// Someone is waiting for the ACK
			default:
				// Nobody is waiting for the data to be sent now (we arrive here too early)
				close(m.receivedACK)
			}
			// fmt.Printf("[MAC%x] ACK Notify End %v\n", m.Address, r)
		} else {
			fmt.Printf("[MAC%x] ACK for packet %d is not expected, expected %d\n", m.Address, header.Index, m.expectedIndex)
			// panic("ACK for packet is not expected")
		}
	}
}

func (m *MACLayer) Send(address MACAddress, data []byte) error {
	// TODO: lock the sending process, so that only one sending process is allowed to call this function at a time
	// packetLength := m.PhysicalLayer.Encoder.Modulator.BytePerFrame - MACHeader{}.NumBytes()
	if m.BytePerFrame == 0 {
		m.BytePerFrame = m.PhysicalLayer.Encoder.Modulator.BytePerFrame - MACHeader{}.NumBytes()
		fmt.Printf("[MAC%x] Payload length is not set, using default value %d", m.Address, m.BytePerFrame)
	}

	// split the data into packets (do not use physical layer's packet splitting)
	packets := make([][]byte, 0)
	header := MACHeader{
		Source:      m.Address,
		Destination: address,
		Type:        MACTypeData,
	}

	for i := 0; i < len(data); i += m.BytePerFrame {
		end := min(i+m.BytePerFrame, len(data))
		if end == len(data) {
			header.IsLast = true
		}
		packet := append(header.ToBytes(), data[i:end]...)
		fmt.Printf("[MAC%x] Making packet %d, length %d\n", m.Address, header.Index, len(packet))
		packets = append(packets, packet)
		header.Index++ // NOTE: this is uint8, so it may overflow
	}

	// send the packets
	for i, packet := range packets {
		retries := 0
	resend:
		for {
			// // wait for the physical layer to be not busy
			// <-m.PowerFreeSignal()
			m.PowerMonitor.Log()
			fmt.Printf("[MAC%x] Sending packet %d\t\n", m.Address, i)

			select {
			case status := <-m.PhysicalLayer.SendAsync(packet):
				fmt.Printf("[MAC%x] Packet %d sent to physical layer status %v\n", m.Address, i, status)

			case err := <-m.PhysicalLayer.DecodeErrorSignal():
				fmt.Printf("[MAC%x] Decode error %v while sending packet %d, possibly due to collision\n\n", m.Address, err, i)
				// Collision detected, resend the packet after a random backoff time
				if m.BackoffTimer == nil {
					fmt.Printf("[MAC%x] No backoff timer, retry immediately\n", m.Address)
				} else {
					backoff := m.BackoffTimer.GetBackoffTime(retries)
					fmt.Printf("[MAC%x] Backoff for %v\n", m.Address, backoff)
					time.Sleep(backoff)
				}
				goto retry
			}

			// wait for the ACK
			fmt.Printf("[MAC%x] is waiting for ACK", m.Address)
			m.expectedIndex = uint8(i)
			select {
			case _, ok := <-m.receivedACK:
				if !ok {
					// Channel was closed since the ACK was detected before we are waiting it, reopen the channel
					m.receivedACK = make(chan struct{})
				}
				// ACK received
				fmt.Printf("[MAC%x] Packet %d sent and ACK received\n", m.Address, i)
				break resend
			case <-time.After(m.ACKTimeout):
				// ACK timeout
				fmt.Printf("[MAC%x] Packet %d ACK timeout\n", m.Address, i)
				goto retry
			}

		retry:
			{

				if retries >= m.MaxRetries {
					return fmt.Errorf("packet %d ACK timeout", i)
				} else {
					retries++
					fmt.Printf("[MAC%x] Resending packet %d, retry %d\n", m.Address, i, retries)
					continue resend
				}
			}

		}

	}

	return nil

}

func (m *MACLayer) SendAsync(address MACAddress, data []byte) <-chan error {
	return async.Promise(func() error { return m.Send(address, data) })
}

func (m *MACLayer) Receive() []byte {
	return <-m.ReceiveAsync()
}

func (m *MACLayer) ReceiveAsync() <-chan []byte {
	return m.OutputChan
}

func (m *MACLayer) ReceiveWithTimeout(timeout time.Duration) ([]byte, error) {
	select {
	case packet := <-m.ReceiveAsync():
		return packet, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("receive timeout")
	}
}
