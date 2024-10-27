package layer

import (
	"Aethernet/pkg/modem"
)

type PhysicalLayer struct {
	device Device

	decoder Decoder
	encoder Encoder
}

type DecodeState int

type Decoder struct {
	demodulator modem.Demodulator
}

type Encoder struct {
	modulator modem.Modulator

	outputBuffer chan []int32 // data to be sent
	current      []int32      // current sending data
}

func (p *PhysicalLayer) Send(data []byte) {
	p.encoder.send(data)
}

func (p *PhysicalLayer) Receive() []byte {
	return <-p.decoder.demodulator.OutputChan
}

func (p *PhysicalLayer) Open() {
	p.device.Start(func(in, out [][]int32) {
		p.inputCallback(in[0])
		p.outputCallback(out[0])
	})
}

func (p *PhysicalLayer) Close() {
	p.device.Stop()
}

func (p *PhysicalLayer) inputCallback(in []int32) {
	in_copy := make([]int32, len(in))
	copy(in_copy, in)
	go p.decoder.read(in_copy)
}

func (p *PhysicalLayer) outputCallback(out []int32) {
	p.encoder.write(out)
}

// try to decode the input and put the result into the Buffer which shares with the some other goroutines
func (d *Decoder) read(in []int32) {
	d.demodulator.Demodulate(in)
}

// try to consume the outputBuffer and write some data to out
func (e *Encoder) write(out []int32) {

	if e.current == nil {
		select {
		case e.current = <-e.outputBuffer:
		default:
			// do nothing
		}
	}

	i := 0
	if e.current != nil {
		// fmt.Printf("[Consumer] len(p.current): %d\n", len(p.current))

		i = copy(out, e.current)
		e.current = e.current[i:]

		if len(e.current) == 0 {
			e.current = nil
			// fmt.Printf("[Consumer] p.current is nil\n")
		}
	}

	for i < len(out) {
		out[i] = 0
		i += 1
	}
}

func (e *Encoder) send(data []byte) {
	e.outputBuffer <- e.modulator.Modulate(data)
}
