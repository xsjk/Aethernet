package layer

import (
	"Aethernet/pkg/device"
	"Aethernet/pkg/modem"
	"fmt"
)

type PhysicalLayer struct {
	Device device.Device

	Decoder Decoder
	Encoder Encoder
}

type DecodeState int

type Decoder struct {
	Demodulator modem.Demodulator
	BufferSize  int

	buffer chan []int32 // data received from the device and to be decoded
}

func (d *Decoder) Mainloop() {
	for in := range d.buffer {
		d.read(in)
	}
}

func (d *Decoder) Init() {
	d.Demodulator.Reset()
	if d.BufferSize == 0 {
		d.BufferSize = 1
	}
	d.buffer = make(chan []int32, d.BufferSize)
}

type EncoderFrame struct {
	Data []int32
	Done chan struct{}
}

type Encoder struct {
	Modulator  modem.Modulator
	BufferSize int

	buffer  chan EncoderFrame // data to be sent
	current *EncoderFrame     // current sending data
}

func (e *Encoder) Init() {
	if e.BufferSize == 0 {
		e.BufferSize = 1
	}
	e.buffer = make(chan EncoderFrame, e.BufferSize)
}

func (p *PhysicalLayer) Send(data []byte) {
	p.Encoder.send(data)
}

func (p *PhysicalLayer) Receive() []byte {
	return <-p.Decoder.Demodulator.OutputChan
}

func (p *PhysicalLayer) Open() {
	p.Decoder.Init()
	p.Encoder.Init()
	p.Device.Start(func(in, out []int32) {
		p.inputCallback(in)
		p.outputCallback(out)
	})
	go p.Decoder.Mainloop()
}

func (p *PhysicalLayer) Close() {
	p.Device.Stop()
}

func (p *PhysicalLayer) inputCallback(in []int32) {
	in_copy := make([]int32, len(in))
	copy(in_copy, in)
	select {
	case p.Decoder.buffer <- in_copy:
	default:
		fmt.Println("[PhysicalLayer] inputBuffer is full")
		panic("inputBuffer is full")
		// TODO: expand the buffer
	}
}

func (p *PhysicalLayer) outputCallback(out []int32) {
	p.Encoder.write(out)
}

// try to decode the input and put the result into the Buffer which shares with the some other goroutines
func (d *Decoder) read(in []int32) {
	d.Demodulator.Demodulate(in)
}

func (e *Encoder) fetch() {
	select {
	case current := <-e.buffer:
		e.current = &current
	default:
		// no new data
	}
}

// try to consume the outputBuffer and write some data to out
func (e *Encoder) write(out []int32) {

	if e.current == nil {
		e.fetch()
	}

	i := 0
	for e.current != nil {
		// fmt.Printf("[Consumer] len(p.current): %d\n", len(p.current))

		j := copy(out[i:], e.current.Data)
		e.current.Data = e.current.Data[j:]

		if len(e.current.Data) == 0 {
			// notify the sender that the data has been sent
			close(e.current.Done)
			e.current = nil
			e.fetch()
		}

		i += j
		if i == len(out) {
			return
		}
	}

	for i < len(out) {
		out[i] = 0
		i += 1
	}
}

// send the data to the device
// the function will block until the data is fully sent
func (e *Encoder) send(data []byte) {
	done := make(chan struct{})
	e.buffer <- EncoderFrame{
		Data: e.Modulator.Modulate(data),
		Done: done,
	}
	<-done
}
