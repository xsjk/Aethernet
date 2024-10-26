package modem

import (
	"Aethernet/pkg/fixed"
	"fmt"
	"sync"
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
	Preamble      []int32
	CarrierSize   int // number of ticks used to represent a bit
	BytePerFrame  int // number of bytes per frame
	FrameInterval int // number of ticks as interval between frames

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
	Preamble    []int32
	CarrierSize int // the size of the carrier

	CorrectionThreshold      fixed.T
	DemodulatePowerThreshold fixed.T
	OutputChan               chan []byte

	once sync.Once

	demodulateState DemodulateStateEnum

	// preamble detection
	currentWindow              []int32
	frameToDecode              []int32 // since preamble is late detected, we need to store the frames when confirming a potential start of the signal for later data extraction
	localMaxPower              fixed.T
	distanceFromPotentialStart int
	potentialHistory           []fixed.T
	correctionFlag             bool

	// data extraction
	crcChecker   CRC8Checker
	rDataEncoded struct {
		data  bitSet[uint16]
		count int
	}
	header struct {
		done bool
		size int
	}
	currentChunk  []byte
	currentPacket []byte

	dataExtractionState DataExtractionStateEnum
	carrierTick         int     // the current carrier tick [0, len(carrier)]
	sum1                fixed.T // sum of the product of the current sample and the current carrier
	sum                 fixed.T
}

func (d *Demodulator) Init() {
	d.demodulateState = preambleDetection

	d.currentWindow = make([]int32, 0)
	d.localMaxPower = fixed.Zero
	d.distanceFromPotentialStart = -1
	d.potentialHistory = make([]fixed.T, 0)
	d.correctionFlag = false

	d.frameToDecode = make([]int32, 0)
	d.crcChecker.Reset()
	d.rDataEncoded.data.Value = 0
	d.rDataEncoded.count = 0
}

func (m *Modulator) Modulate(inputBytes []byte) []int32 {

	frameCount := (len(inputBytes) + m.BytePerFrame - 1) / m.BytePerFrame
	samplePerBit := m.CarrierSize

	modulatedData := make([]int32, 0, frameCount*(len(m.Preamble)+(1+m.BytePerFrame+1)*10*samplePerBit+m.FrameInterval))

	modulateBit := func(bit bool) {
		for range m.CarrierSize {
			if bit {
				modulatedData = append(modulatedData, -0x7FFFFFFF)
			} else {
				modulatedData = append(modulatedData, 0x7FFFFFFF)
			}
		}
	}

	for i := 0; i < frameCount; i++ {
		bytes := inputBytes[i*m.BytePerFrame : min((i+1)*m.BytePerFrame, len(inputBytes))]

		// add the preamble
		modulatedData = append(modulatedData, m.Preamble...)

		// add the header
		header := byte(len(bytes))
		if i == frameCount-1 {
			header |= 0b10000000
		}
		BitSet(B8B10[header]).ForEach(modulateBit, 10)

		// modulate the data
		m.crcChecker.Reset()
		for _, b := range bytes {
			m.crcChecker.Update(b)
			BitSet(B8B10[b]).ForEach(modulateBit, 10)
		}
		crcByte := m.crcChecker.Get()

		// modulate the CRC8 byte
		BitSet(B8B10[crcByte]).ForEach(modulateBit, 10)

		fmt.Println("[Modulation] CRC8:", ByteToBool([]byte{crcByte}))

		// add the interval
		for j := 0; j < m.FrameInterval; j++ {
			modulatedData = append(modulatedData, 0)
		}

	}

	return modulatedData
}

func (d *Demodulator) Demodulate(inputSignal []int32) {
	d.once.Do(d.Init)

	for i, currentSample := range inputSignal {
		debugIndex = i
		d.Update(currentSample)
	}
}

var debugIndex int

func (d *Demodulator) Update(currentSample int32) {

	switch d.demodulateState {
	case preambleDetection:
		d.detectPreamble(currentSample)
	case dataExtraction:
		d.extractData(currentSample)
	}
}

func (d *Demodulator) detectPreamble(currentSample int32) {

	d.currentWindow = append(d.currentWindow, currentSample)

	if len(d.currentWindow) < len(d.Preamble) {
		return
	}

	power := dotProduct(d.currentWindow, d.Preamble)
	d.currentWindow = d.currentWindow[1:]

	// find a potential start of the signal
	if power > d.localMaxPower && power > d.DemodulatePowerThreshold {
		fmt.Printf("[Demodulation] find a potential start of the signal at %v where power: %.2f\n", debugIndex, fixed.T(power).Float())
		d.localMaxPower = power
		d.frameToDecode = d.frameToDecode[:0]
		d.distanceFromPotentialStart = 0
		d.potentialHistory = append(d.potentialHistory, power)
	} else if d.distanceFromPotentialStart == -1 {
		// potential start of the signal is not found yet
	} else {
		// append the currentSample to the frameToDecode if necessary
		d.frameToDecode = append(d.frameToDecode, currentSample)
		d.distanceFromPotentialStart += 1
	}

	// a real start of the signal is found
	if d.distanceFromPotentialStart >= len(d.Preamble) {

		fmt.Printf("[Demodulation] find the start of the signal at %v where power: %.2f\n", debugIndex-d.distanceFromPotentialStart, fixed.T(d.localMaxPower).Float())
		fmt.Println("[Demodulation] potentialHistory:", d.potentialHistory)

		// determine whether to flip
		d.correctionFlag = false
		if len(d.potentialHistory) > 1 {
			lastPotentialStart := d.potentialHistory[len(d.potentialHistory)-1]
			secondLastPotentialStart := d.potentialHistory[len(d.potentialHistory)-2]
			increaseRate := lastPotentialStart.Sub(secondLastPotentialStart).Div(secondLastPotentialStart)
			fmt.Printf("[Demodulation] increaseRate: %.2f\n", fixed.T(increaseRate).Float())

			d.correctionFlag = increaseRate < d.CorrectionThreshold
		} else {
			fmt.Println("[Demodulation] not enough potentialHistory to determine correction, you may decrease the POWER_THRESHOLD")
		}

		fmt.Printf("[Demodulation] correctionFlag: %v\n", d.correctionFlag)

		d.localMaxPower = 0
		d.currentWindow = d.currentWindow[:0]
		d.distanceFromPotentialStart = -1
		d.demodulateState = dataExtraction
		d.rDataEncoded.data.Value = 0
		d.rDataEncoded.count = 0
		for _, sample := range d.frameToDecode {
			if d.demodulateState == dataExtraction {
				d.extractData(sample)
			} else {
				break
			}
		}
		d.frameToDecode = d.frameToDecode[:0]
	}
}

func (d *Demodulator) extractData(currentSample int32) {

	d.sum += fixed.T(currentSample >> fixed.N)
	d.carrierTick += 1

	if d.carrierTick%d.CarrierSize > 0 {
		return
	}

	if d.rDataEncoded.count >= 16 {
		panic("Data is too long")
	}

	if d.sum < 0 {
		d.rDataEncoded.data.Set(d.rDataEncoded.count)
	}
	d.rDataEncoded.count += 1

	d.sum = 0
	d.sum1 = 0
	d.carrierTick = 0

	if d.rDataEncoded.count < 10 {
		return
	}

	currentByte, exists := B10B8[d.rDataEncoded.data.Value]
	if !exists {
		fmt.Printf("[Demodulation] Warning: B10B8 does not contain key %v\n", d.rDataEncoded.data.Value)
		d.rDataEncoded.data.Value = 0
		d.rDataEncoded.count = 0
		d.demodulateState = preambleDetection
		return
	}
	d.rDataEncoded.data.Value = 0
	d.rDataEncoded.count = 0

	switch d.dataExtractionState {
	case receiveHeader:
		d.receiveHeader(currentByte)
	case receiveData:
		d.receiveData(currentByte)
	case receiveCRC:
		d.receiveCRC(currentByte)
	}
}

func (d *Demodulator) receiveHeader(currentSample byte) {
	d.header.done = currentSample&0b10000000 != 0
	d.header.size = int(currentSample & 0b01111111)

	if d.header.size == 0 { // invalid packet
		fmt.Println("[Demodulation] Warning: header.size is 0, invalid packet")
		d.demodulateState = preambleDetection
		return
	}

	// prepare for receiving data
	d.crcChecker.Reset()
	d.dataExtractionState = receiveData
	if len(d.currentChunk) != 0 {
		panic("rDataDecoded is not empty")
	}
}

func (d *Demodulator) receiveData(currentSample byte) {
	d.currentChunk = append(d.currentChunk, currentSample)
	d.crcChecker.Update(currentSample)
	if len(d.currentChunk) == d.header.size { // the packet is fully received
		d.dataExtractionState = receiveCRC
	}
}

func (d *Demodulator) receiveCRC(currentSample byte) {
	if d.crcChecker.Get() == currentSample {
		fmt.Println("[Demodulation] CRC8 check passed")
	} else {
		fmt.Println("[Demodulation] CRC8 check failed")
	}

	d.currentPacket = append(d.currentPacket, d.currentChunk...)
	if d.header.done {
		d.OutputChan <- d.currentPacket
		d.currentPacket = []byte{}
	}
	d.currentChunk = d.currentChunk[:0]
	d.demodulateState = preambleDetection
	d.dataExtractionState = receiveHeader
	d.header.done = false
	d.header.size = 0

}
