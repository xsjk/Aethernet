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
	// if rand.Intn(2) == 0 {
	// 	return b.MinDelay
	// } else {
	// 	return b.MaxDelay
	// }
}

type MACLayer struct {
	PhysicalLayer

	BytePerFrame int
	Address      MACAddress
	ACKTimeout   time.Duration
	MaxRetries   int
	BackoffTimer BackoffTimer
	BufferSize   int

	// Send
	expectedIndex uint8
	receivedACK   chan uint8

	// Receive
	currentPacket []byte
	outputChan    chan []byte
}

func (m *MACLayer) Open() {
	m.PhysicalLayer.Open()
	m.receivedACK = make(chan uint8, 1)
	m.outputChan = make(chan []byte, m.BufferSize)
	go func() {
		for packet := range m.PhysicalLayer.ReceiveAsync() {
			header := MACHeader{}
			header.FromBytes(packet[:header.NumBytes()])
			if header.Destination == m.Address {
				if len(packet) < header.NumBytes() {
					fmt.Printf("[MAC%x] Packet is too short, dropping\n", m.Address)
					panic("Packet is too short")
				}
				m.handle(header, packet[header.NumBytes():])
			}
		}
	}()
}

func (m *MACLayer) sendACK(address MACAddress, index uint8) {
	// <-m.PowerFreeSignal()
	m.PhysicalLayer.Send(MACHeader{
		Source:      m.Address,
		Destination: address,
		Type:        MACTypeACK,
		Index:       index,
	}.ToBytes())
	fmt.Printf("[MAC%x] ACK for packet %d sent\n", m.Address, index)
}

func (m *MACLayer) handle(header MACHeader, data []byte) {

	switch header.Type {
	case MACTypeData:
		if header.Index == m.expectedIndex {
			m.currentPacket = append(m.currentPacket, data...)
			fmt.Printf("[MAC%x] Append packet %d, length %d, total %d\n", m.Address, header.Index, len(data), len(m.currentPacket))
			m.expectedIndex++
			if header.IsLast {
				m.expectedIndex = 0
				select {
				case m.outputChan <- m.currentPacket:
					fmt.Printf("[MAC%x] Packet %d received\n", m.Address, header.Index)
				default:
					fmt.Printf("[MAC%x] Output channel is full\n", m.Address)
					panic("Output channel is full")
				}
				m.currentPacket = nil
			}
			go m.sendACK(header.Source, header.Index)
		} else if header.Index == m.expectedIndex-1 {
			fmt.Printf("[MAC%x] Packet %d is a duplicate, resending ACK\n", m.Address, header.Index)
			go m.sendACK(header.Source, header.Index)
		} else {
			fmt.Printf("[MAC%x] Packet %d is not expected, expected %d\n", m.Address, header.Index, m.expectedIndex)
		}
	case MACTypeACK:
		// check the index with the current sending packet
		select {
		case m.receivedACK <- header.Index:
			// Someone is waiting for the ACK
		default:
			for i := 0; i < len(m.receivedACK); i++ {
				index := <-m.receivedACK
				fmt.Printf("[MAC%x] ACK channel is full, dropping packet %d\n", m.Address, index)
				if index != header.Index {
					m.receivedACK <- index
				}
			}
			select {
			case m.receivedACK <- header.Index:
			default:
				panic("ACK channel is full")
			}
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
		backoff := time.Duration(0)

	resend:

		for {
			var ackReceived chan struct{}
			var ackStopListening chan struct{}
			var stopListening chan struct{}

			// // wait for the physical layer to be not busy
			// <-m.PowerFreeSignal()
			m.PowerMonitor.Log()
			fmt.Printf("[MAC%x] Sending packet %d\t\n", m.Address, i)

			m.PhysicalLayer.Decoder.Demodulator.ClearErrorSignal()
			select {
			case status := <-m.PhysicalLayer.SendAsync(packet):
				fmt.Printf("[MAC%x] Packet %d sent to physical layer status %v\n", m.Address, i, status)

			case err := <-m.PhysicalLayer.DecodeErrorSignal():
				fmt.Printf("[MAC%x] Decode error %v while sending packet %d, possibly due to collision\n\n", m.Address, err, i)
				// Collision detected, resend the packet after a random backoff time
				if m.BackoffTimer == nil {
					fmt.Printf("[MAC%x] No backoff timer, retry immediately\n", m.Address)
				} else {
					backoff = m.BackoffTimer.GetBackoffTime(retries)
				}
				m.PhysicalLayer.CancelSend()
				goto retry
			}

			// wait for the ACK
			fmt.Printf("[MAC%x] is waiting for ACK of packet %d\n", m.Address, i)

			ackReceived = make(chan struct{})
			ackStopListening = make(chan struct{})
			stopListening = make(chan struct{})

			go func() {
				for {
					select {
					case index := <-m.receivedACK:
						if index == uint8(i) {
							ackReceived <- struct{}{}
							close(ackStopListening)
							return
						} else {
							fmt.Printf("[MAC%x] ACK for packet %d is not expected, expected %d\n", m.Address, index, i)
						}
					case ackStopListening <- struct{}{}:
						return
					case <-stopListening:
						return
					}
				}
			}()

			m.PhysicalLayer.Decoder.Demodulator.ClearErrorSignal()

			select {
			case <-ackReceived:
				// ACK received
				<-ackStopListening
				fmt.Printf("[MAC%x] Packet %d ACK received\n", m.Address, i)
				break resend
			case <-time.After(m.ACKTimeout):
				// ACK timeout
				<-ackStopListening
				fmt.Printf("[MAC%x] Packet %d ACK timeout retry %d\n", m.Address, i, retries)
				goto retry

			case err := <-m.PhysicalLayer.DecodeErrorSignal():
				fmt.Printf("[MAC%x] Decode error %v while waiting for ack of %d, possibly due to collision\n\n", m.Address, err, i)
				// Collision detected, resend the packet after a random backoff time
				if m.BackoffTimer == nil {
					fmt.Printf("[MAC%x] No backoff timer, retry immediately\n", m.Address)
				} else {
					backoff = m.BackoffTimer.GetBackoffTime(retries)
				}
				m.PhysicalLayer.CancelSend()
				goto retry
			}

		retry:

			if stopListening != nil {
				close(stopListening)
			}

			{
				if retries >= m.MaxRetries {
					return fmt.Errorf("packet %d ACK timeout after %d retries", i, m.MaxRetries)
				} else {
					if backoff != 0 {
						fmt.Printf("[MAC%x] Backoff for %v\n", m.Address, backoff)
						<-time.After(backoff)
						backoff = 0
					}
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
	return m.outputChan
}

func (m *MACLayer) ReceiveWithTimeout(timeout time.Duration) ([]byte, error) {
	select {
	case packet := <-m.ReceiveAsync():
		return packet, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("receive timeout")
	}
}
