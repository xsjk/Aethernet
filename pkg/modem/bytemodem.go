package modem

import (
	"Aethernet/pkg/async"
	"Aethernet/pkg/fixed"
	"fmt"
	"sync"
	"time"
)

const (
	AJUST_THRESHOLD = fixed.Zero

	HEADER_SIZE = 2
)

type ByteModem interface {
	Modulate(inputBytes []byte) []int32
	Demodulate(inputSignal []int32) []byte
}

type NaiveByteModem struct {
	Modulator
	Demodulator
}

type Modulator struct {
	Preamble             []int32
	CarrierSize          int // number of ticks used to represent a bit
	CarrierSizeForHeader int
	BytePerFrame         int // number of bytes per frame
	FrameInterval        int // number of ticks as interval between frames
	Amplitude            int32

	crcChecker CRC8Checker
}

type DemodulateStateEnum int

const (
	preambleDetection DemodulateStateEnum = iota
	dataExtraction
)

type DataExtractionStateEnum int

const (
	receiveHeader DataExtractionStateEnum = iota
	receiveData
	receiveCRC
)

type Demodulator struct {
	Preamble                 []int32
	CarrierSize              int // the size of the carrier
	CarrierSizeForHeader     int // the size of the carrier for the header
	BufferSize               int // the size of the buffer for the output channel
	DemodulatePowerThreshold fixed.T

	outputChan  chan []byte // demodulated data will be sent to this channel, the channel has no buffer, so the receiver must be ready to receive the data
	errorSignal async.Signal[error]

	once sync.Once

	demodulateState DemodulateStateEnum

	// preamble detection
	currentWindow              []int32
	frameToDecode              []int32 // since preamble is late detected, we need to store the frames when confirming a potential start of the signal for later data extraction
	powerPrev                  fixed.T
	lastPoppedSample           int32
	localMaxPower              fixed.T
	localMaxPowerPrev          fixed.T
	adjustment                 fixed.T
	resampler                  AdjustmentResampler
	distanceFromStart          int
	distanceFromPotentialStart int
	powerHistory               []fixed.T

	// data extraction
	crcChecker  CRC8Checker
	currentBits struct {
		data  bitSet[uint16]
		count int
	}
	currentHeader struct {
		done  bool
		size  int
		index int
	}
	currentChunk  []byte
	currentPacket []byte

	dataExtractionState DataExtractionStateEnum
	carrierTick         int     // the current carrier tick [0, len(carrier)]
	sum                 fixed.T // sum of the product of the current sample and the current carrier

}

type AdjustmentResampler struct {
	LastSample fixed.T
	P          fixed.T
}

func interpolate(a, b, t fixed.T) fixed.T {
	return a.Mul(fixed.One - t).Add(b.Mul(t))
}

func (a *AdjustmentResampler) Update(currentSample fixed.T) (outputSample fixed.T) {
	lastSample := a.LastSample
	a.LastSample = currentSample
	outputSample = interpolate(lastSample, currentSample, a.P)
	return
}

func (d *Demodulator) Init() {
	d.errorSignal = make(chan error)
	d.outputChan = make(chan []byte, d.BufferSize)
}

func (d *Demodulator) Reset() {
	d.demodulateState = preambleDetection

	d.currentWindow = make([]int32, 0)
	d.frameToDecode = make([]int32, 0)
	d.powerPrev = fixed.Zero
	d.localMaxPower = fixed.Zero
	d.localMaxPowerPrev = fixed.Zero
	d.distanceFromPotentialStart = -1
	d.powerHistory = make([]fixed.T, 0)

	d.crcChecker.Reset()
	d.currentBits.data.Value = 0
	d.currentBits.count = 0
	d.currentHeader.done = false
	d.currentHeader.size = 0
	d.currentChunk = make([]byte, 0)
	d.currentPacket = make([]byte, 0)
	d.dataExtractionState = receiveHeader
	d.carrierTick = 0
	d.sum = fixed.Zero
}

func (m Modulator) Modulate(inputBytes []byte) []int32 {

	frameCount := (len(inputBytes) + m.BytePerFrame - 1) / m.BytePerFrame

	if m.CarrierSizeForHeader == 0 {
		debugLog("[Modulation] Warning: CarrierSizeForHeader is not set, using CarrierSize\n")
		m.CarrierSizeForHeader = max(m.CarrierSize, 2)
	}

	modulatedData := make([]int32, 0, frameCount*
		(len(m.Preamble)+
			10*(m.CarrierSizeForHeader*HEADER_SIZE+
				m.CarrierSize*(m.BytePerFrame+1))+
			m.FrameInterval))

	if m.Amplitude == 0 {
		fmt.Printf("[Modulation] Warning: Amplitude is not set, using 0x7FFFFFFF\n")
		m.Amplitude = 0x7FFFFFFF
	}

	var samplePerBit int
	modulateBit := func(bit bool) {
		for range samplePerBit {
			if bit {
				modulatedData = append(modulatedData, -m.Amplitude)
			} else {
				modulatedData = append(modulatedData, m.Amplitude)
			}
		}
	}

	for i := 0; i < frameCount; i++ {
		bytes := inputBytes[i*m.BytePerFrame : min((i+1)*m.BytePerFrame, len(inputBytes))]

		// add the preamble
		modulatedData = append(modulatedData, m.Preamble...)

		// add the header
		if len(bytes) > 127 {
			panic("Data is too long to fit in the header")
		}
		if i > 255 {
			panic("Frame count is too large to fit in the header")
		}
		header := make([]byte, HEADER_SIZE)
		header[0] = byte(len(bytes))
		if i == frameCount-1 {
			header[0] |= 0b10000000
		}
		header[1] = byte(i)
		samplePerBit = m.CarrierSizeForHeader
		for _, b := range header {
			BitSet(B8B10[b]).ForEach(modulateBit, 10)
		}

		samplePerBit = m.CarrierSize

		// modulate the data
		m.crcChecker.Reset()
		for _, b := range bytes {
			m.crcChecker.Update(b)
			BitSet(B8B10[b]).ForEach(modulateBit, 10)
		}
		crcByte := m.crcChecker.Get()

		// modulate the CRC8 byte
		BitSet(B8B10[crcByte]).ForEach(modulateBit, 10)

		// add the interval
		for j := 0; j < m.FrameInterval; j++ {
			modulatedData = append(modulatedData, 0)
		}

	}

	return modulatedData
}

func (d *Demodulator) Demodulate(inputSignal []int32) (err error) {
	d.once.Do(d.Reset)

	for _, currentSample := range inputSignal {
		err = d.Update(currentSample)
		if err != nil {
			debugLog("[Demodulation] Error: %v at %v\n", err, d.distanceFromStart)
			d.signalError(err)
		}
	}
	return
}

func (d *Demodulator) Update(currentSample int32) (err error) {

	switch d.demodulateState {
	case preambleDetection:
		err = d.detectPreamble(currentSample)
	case dataExtraction:
		err = d.extractData(currentSample)
	}

	return
}

func estimateAdjustment(pl, pm, pr fixed.T) (adj fixed.T) {
	if pl > pr {
		adj = fixed.One - (pm + pr).Div(pm+pl)
	} else {
		adj = (pm + pl).Div(pm+pr) - fixed.One
	}
	// fmt.Printf("Estimate adjustment with %.3f, %.3f, %.3f: %.3f\n", pl.Float(), pm.Float(), pr.Float(), adj.Float())
	return
}

func (d *Demodulator) detectPreamble(currentSample int32) (err error) {

	d.currentWindow = append(d.currentWindow, currentSample)

	if len(d.currentWindow) < len(d.Preamble) {
		return
	}

	power := dotProduct(d.currentWindow, d.Preamble)
	d.powerHistory = append(d.powerHistory, power)

	poppedSample := d.currentWindow[0]
	d.currentWindow = d.currentWindow[1:]

	// find a potential start of the signal
	if power > d.localMaxPower && power > d.DemodulatePowerThreshold {
		debugLog("[Demodulation] find a potential start of the signal where power: %.2f\n", fixed.T(power).Float())
		d.localMaxPower = power
		d.localMaxPowerPrev = d.powerPrev - fixed.T((int64(d.lastPoppedSample)*int64(d.Preamble[0]))>>(31+fixed.N))
		d.frameToDecode = d.frameToDecode[:0]
		d.distanceFromPotentialStart = 0

		// this sample is the end of the preamble but is still needed for the data extraction because of potential adjustment
		d.frameToDecode = append(d.frameToDecode, currentSample)
	} else if d.distanceFromPotentialStart == -1 {
		// potential start of the signal is not found yet
	} else {
		// append the currentSample to the frameToDecode if necessary
		d.frameToDecode = append(d.frameToDecode, currentSample)
		if d.distanceFromPotentialStart == 0 {

			dotProductLatter := power - fixed.T((int64(d.currentWindow[len(d.Preamble)-2])*int64(d.Preamble[len(d.Preamble)-1]))>>(31+fixed.N))
			d.adjustment = estimateAdjustment(d.localMaxPowerPrev, d.localMaxPower, dotProductLatter)
			if d.adjustment > AJUST_THRESHOLD {
				d.adjustment = -AJUST_THRESHOLD
			} else if d.adjustment < -AJUST_THRESHOLD {
				d.adjustment = AJUST_THRESHOLD
			}
			if d.adjustment > 0 {
				d.resampler.P = d.adjustment
			} else {
				d.resampler.P = -d.adjustment
			}
			d.resampler.LastSample = fixed.T(d.frameToDecode[1] >> fixed.N)
			d.frameToDecode = d.frameToDecode[2:]
			if d.resampler.P < 0 || d.resampler.P > fixed.One {
				panic("Invalid adjustment")
			}

		}
		d.distanceFromPotentialStart += 1
	}
	d.powerPrev = power
	d.lastPoppedSample = poppedSample

	// a real start of the signal is found
	if d.distanceFromPotentialStart >= len(d.Preamble) {

		debugLog("[Demodulation] find the start of the signal where adjustment %.2f\n", d.adjustment.Float())

		d.distanceFromStart = 0

		// determine whether to flip
		d.localMaxPower = 0
		d.currentWindow = d.currentWindow[:0]
		d.distanceFromPotentialStart = -1
		d.demodulateState = dataExtraction
		d.currentBits.data.Value = 0
		d.currentBits.count = 0
		for _, sample := range d.frameToDecode {
			if d.demodulateState == dataExtraction {
				err = d.extractData(sample)
				if err != nil {
					return
				}
			} else {
				break
			}
		}
		d.frameToDecode = d.frameToDecode[:0]
		d.powerHistory = d.powerHistory[:0]
	}
	return
}

func (d *Demodulator) extractData(currentSample int32) (err error) {

	d.distanceFromStart++
	// isLastTick := d.distanceFromStart == d.currentHeader.size*10

	if d.CarrierSizeForHeader == 0 {
		debugLog("[Modulation] Warning: CarrierSizeForHeader is not set, using CarrierSize\n")
		d.CarrierSizeForHeader = max(d.CarrierSize, 2)
	}

	cur := d.resampler.Update(fixed.T(currentSample >> fixed.N))

	// expectLength := ((d.currentHeader.size+1)*d.CarrierSize + 1*d.CarrierSizeForHeader) * 10
	// fmt.Printf("Extract data %d/%d: %f\n", d.distanceFromStart, expectLength, cur.Float())

	d.sum += cur
	d.carrierTick += 1

	var samplePerBit int
	switch d.dataExtractionState {
	case receiveHeader:
		samplePerBit = d.CarrierSizeForHeader
	case receiveData:
		samplePerBit = d.CarrierSize
	case receiveCRC:
		samplePerBit = d.CarrierSize
	}

	if d.carrierTick%samplePerBit > 0 {
		return
	}

	if d.currentBits.count >= 16 {
		panic("Data is too long")
	}

	if d.sum < 0 {
		d.currentBits.data.Set(d.currentBits.count)
	}
	d.currentBits.count += 1

	d.sum = 0
	d.carrierTick = 0

	if d.currentBits.count < 10 {
		return
	}

	currentByte, exists := B10B8[d.currentBits.data.Value]
	if !exists {
		err = fmt.Errorf("B10B8 does not contain key %v", d.currentBits.data.Value)
		d.currentBits.data.Value = 0
		d.currentBits.count = 0
		d.demodulateState = preambleDetection
		d.currentPacket = []byte{}
		return
	}
	d.currentBits.data.Value = 0
	d.currentBits.count = 0

	switch d.dataExtractionState {
	case receiveHeader:
		err = d.receiveHeader(currentByte)
	case receiveData:
		err = d.receiveData(currentByte)
	case receiveCRC:
		err = d.receiveCRC(currentByte)
	}

	if d.demodulateState == preambleDetection {
		err = d.detectPreamble(currentSample)
	}

	return
}

func (d *Demodulator) receiveHeader(currentSample byte) (err error) {
	d.currentChunk = append(d.currentChunk, currentSample)
	if len(d.currentChunk) < HEADER_SIZE {
		return
	}
	defer func() {
		d.currentChunk = d.currentChunk[:0]
	}()

	currentIndex := int(d.currentChunk[1])
	if currentIndex == 0 {
		// a new packet is detected
	} else if currentIndex != d.currentHeader.index+1 {
		// the current packet is not following the previous packet
		err = fmt.Errorf("current index %d is not the expected index %d", currentIndex, d.currentHeader.index+1)
		d.demodulateState = preambleDetection
		return
	}
	d.currentHeader.done = d.currentChunk[0]&0b10000000 != 0
	d.currentHeader.size = int(d.currentChunk[0] & 0b01111111)
	d.currentHeader.index = int(d.currentChunk[1])
	if d.currentHeader.done {
		debugLog("[Demodulation] Last packet got\n")
	}

	if d.currentHeader.size == 0 { // invalid packet
		err = fmt.Errorf("header.size is 0, invalid packet")
		d.demodulateState = preambleDetection
		return
	}

	// prepare for receiving data
	d.crcChecker.Reset()
	d.dataExtractionState = receiveData
	return
}

func (d *Demodulator) receiveData(currentSample byte) (err error) {
	d.currentChunk = append(d.currentChunk, currentSample)
	d.crcChecker.Update(currentSample)
	if len(d.currentChunk) == d.currentHeader.size { // the packet is fully received
		d.dataExtractionState = receiveCRC
	}
	return
}

func (d *Demodulator) receiveCRC(currentSample byte) (err error) {
	crcOK := d.crcChecker.Get() == currentSample
	if crcOK {
		d.currentPacket = append(d.currentPacket, d.currentChunk...)
		debugLog("[Demodulation] CRC8 check passed length %d\n", len(d.currentPacket))
		if d.currentHeader.done {
			select {
			case d.outputChan <- d.currentPacket:
			case <-time.After(1 * time.Second):
				panic("outputChan is not consumed")
				// default:
				// 	debugLog("[Demodulation] Warning: outputChan is full, dropping packet\n")
				// 	<-d.outputChan
				// 	d.outputChan <- d.currentPacket
			}
			d.currentPacket = []byte{}
		}
	} else {
		err = fmt.Errorf("CRC8 check failed")
	}

	d.currentChunk = d.currentChunk[:0]
	d.demodulateState = preambleDetection
	d.dataExtractionState = receiveHeader
	d.currentHeader.done = false
	d.currentHeader.size = 0
	return
}

func (d *Demodulator) signalError(err error) {
	if d.errorSignal == nil {
		panic("errorSignal is nil")
	}

	// Signal the decode error
	select {
	case d.errorSignal <- err:
	case <-d.errorSignal:
		debugLog("[Demodulation] Warning: errorSignal is full, dropping error: %v\n", err)
	default:
		debugLog("[Demodulation] Warning: errorSignal is not consumed, dropping error: %v\n", err)
	}
}

func (d *Demodulator) ClearErrorSignal() {
	select {
	case <-d.errorSignal:
	default:
	}
}

func (d *Demodulator) ErrorSignal() <-chan error {
	return d.errorSignal
}

func (d *Demodulator) ReceiveAsync() <-chan []byte {
	return d.outputChan
}
