package layers

import (
	"Aethernet/pkg/async"
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
	Done chan bool
}

type Encoder struct {
	Modulator  modem.Modulator
	BufferSize int

	buffer  chan EncoderFrame // data to be sent
	current *EncoderFrame     // current sending data
}

type PowerMonitor struct {
	Threshold  fixed.T
	WindowSize int

	notBusy async.Signal[struct{}]

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

func (d *Decoder) ErrorSignal() <-chan error {
	return d.Demodulator.ErrorSignal()
}

func (e *Encoder) Init() {
	if e.BufferSize == 0 {
		e.BufferSize = 1
	}
	e.Reset()
}

func (e *Encoder) Reset() {
	e.buffer = make(chan EncoderFrame, e.BufferSize)
	if e.current != nil {
		// TODO notify the sender that the data has been cancelled
		e.current.Done <- false
	}
	e.current = nil
}

func (p *PhysicalLayer) Send(data []byte) {
	<-p.SendAsync(data)
}

func (p *PhysicalLayer) SendAsync(data []byte) <-chan bool {
	return p.Encoder.sendAsync(data)
}

func (p *PhysicalLayer) IsSending() bool {
	return p.Encoder.current != nil || len(p.Encoder.buffer) > 0
}

func (p *PhysicalLayer) DecodeErrorSignal() <-chan error {
	return p.Decoder.ErrorSignal()
}

func (p *PhysicalLayer) CancelSend() {
	p.Encoder.Reset()
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

func (p *PhysicalLayer) PowerFreeSignal() <-chan struct{} {
	return p.PowerMonitor.NotBusySignal()
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
			e.current.Done <- true
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

// sends data to the device and returns a boolean channel to indicate whether the data has been sent or cancelled
func (e *Encoder) sendAsync(data []byte) <-chan bool {
	done := make(chan bool, 1)
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
	if !b.IsBusy() {
		b.notBusy.Notify()
	}
	// fmt.Printf("[PowerMonitor] Power: %.2f, Threshold: %.2f, Busy: %t\n", b.Power.Float(), b.Threshold.Float(), b.IsBusy())
}

func (b *PowerMonitor) NotBusySignal() <-chan struct{} {
	return b.notBusy.Signal()
}

func (b *PowerMonitor) IsBusy() bool {
	return b.Power > b.Threshold
}

func (b *PowerMonitor) Reset() {
	b.latest = nil
	b.sum = 0
	b.Power = 0
}
