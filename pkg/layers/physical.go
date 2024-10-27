package layer

import (
	"Aethernet/pkg/modem"
	"fmt"
)

type PhysicalLayer struct {
	Device Device

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

type Encoder struct {
	Modulator  modem.Modulator
	BufferSize int

	buffer  chan []int32 // data to be sent
	current []int32      // current sending data
}

func (e *Encoder) Init() {
	if e.BufferSize == 0 {
		e.BufferSize = 1
	}
	e.buffer = make(chan []int32, e.BufferSize)
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
	p.Device.Start(func(in, out [][]int32) {
		p.inputCallback(in[0])
		p.outputCallback(out[0])
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
	case e.current = <-e.buffer:
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

		i += copy(out[i:], e.current)
		e.current = e.current[i:]

		if len(e.current) == 0 {
			e.current = nil
			e.fetch()
		}

		if i == len(out) {
			return
		}
	}

	for i < len(out) {
		out[i] = 0
		i += 1
	}
}

func (e *Encoder) send(data []byte) {
	e.buffer <- e.Modulator.Modulate(data)
}
