package layers

import (
	"Aethernet/pkg/device"
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/modem"
	"fmt"
)

type PhysicalLayer struct {
	Device device.Device

	Decoder Decoder
	Encoder Encoder

	PowerMonitor PowerMonitor

	LateUpdate func(in, out []int32)
}

type DecodeState int

type Decoder struct {
	Demodulator modem.Demodulator
	BufferSize  int

	buffer chan []int32 // data received from the device and to be decoded
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

type PowerMonitor struct {
	Threshold   fixed.T
	WindowSize  int
	notBusyChan chan struct{}

	latest []fixed.T
	sum    fixed.T
	Power  fixed.T
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

func (e *Encoder) Init() {
	if e.BufferSize == 0 {
		e.BufferSize = 1
	}
	e.buffer = make(chan EncoderFrame, e.BufferSize)
}

func (p *PhysicalLayer) Send(data []byte) {
	<-p.SendAsync(data)
}

func (p *PhysicalLayer) SendAsync(data []byte) chan struct{} {
	return p.Encoder.sendAsync(data)
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
		p.PowerMonitor.Update(in)
		if p.LateUpdate != nil {
			p.LateUpdate(in, out)
		}
	})
	go p.Decoder.Mainloop()
}

func (p *PhysicalLayer) Close() {
	p.Device.Stop()
}

func (p *PhysicalLayer) inputCallback(in []int32) {
	in_copy := make([]int32, len(in))
	copy(in_copy, in)
	p.Decoder.submit(in_copy)
}

func (p *PhysicalLayer) outputCallback(out []int32) {
	p.Encoder.write(out)
}

func (p *PhysicalLayer) MeasurePower() fixed.T {
	return p.PowerMonitor.Power
}

func (p *PhysicalLayer) IsBusy() bool {
	return p.PowerMonitor.IsBusy()
}

// try to decode the input and put the result into the Buffer which shares with the some other goroutines
func (d *Decoder) read(in []int32) {
	d.Demodulator.Demodulate(in)
}

// submit data to be decoded
func (d *Decoder) submit(data []int32) {
	select {
	case d.buffer <- data:
	default:
		fmt.Println("[Decoder] buffer is full")
		panic("inputBuffer is full")
	}
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
func (e *Encoder) sendAsync(data []byte) chan struct{} {
	done := make(chan struct{})
	e.buffer <- EncoderFrame{
		Data: e.Modulator.Modulate(data),
		Done: done,
	}
	return done
}

func (b *PowerMonitor) Update(in []int32) {

	if b.WindowSize == 0 {
		return
	}
	maxsum := fixed.Zero
	for i := range in {
		v := fixed.T(in[i] >> fixed.N)
		if v < 0 {
			v = -v
		}
		if len(b.latest) >= b.WindowSize {
			b.sum -= b.latest[0]
			b.latest = b.latest[1:]
		}
		b.latest = append(b.latest, v)
		b.sum += v
		if b.sum > maxsum {
			b.Power = b.sum.Div(fixed.FromInt(b.WindowSize))
		}
	}
	if !b.IsBusy() && b.notBusyChan != nil {
		select {
		case <-b.notBusyChan:
		default:
			close(b.notBusyChan)
		}
	}
	// fmt.Printf("[PowerMonitor] Power: %.2f, Threshold: %.2f, Busy: %t\n", b.Power.Float(), b.Threshold.Float(), b.IsBusy())
}

func (b *PowerMonitor) WaitAsync() chan struct{} {
	b.notBusyChan = make(chan struct{})
	return b.notBusyChan
}

func (b *PowerMonitor) IsBusy() bool {
	return b.Power > b.Threshold
}

func (b *PowerMonitor) Reset() {
	b.latest = nil
	b.sum = 0
	b.Power = 0
}
