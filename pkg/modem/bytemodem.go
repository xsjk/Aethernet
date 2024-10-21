package modem

import (
	"Aethernet/pkg/fixed"
	"fmt"
)

type ByteModem interface {
	Modulate(inputBits []byte) []int32
	Demodulate(inputSignal []int32) []byte
}

type NaiveByteModem struct {
	Preamble []int32

	BytePerFrame  int // number of bytes per frame
	FrameInterval int // interval between frames

	CRCChecker CRC8Checker

	Carriers [2][]int32

	CorrectionThreshold      fixed.T
	DemodulatePowerThreshold fixed.T
}

func forEachBit(input byte, f func(bool)) {
	for i := 0; i < 8; i++ {
		f((input>>uint(7-i))&1 == 1)
	}
}

func (m *NaiveByteModem) Modulate(inputBytes []byte) []int32 {

	// frameCount := (len(inputBits) + m.BitPerFrame - 1) / m.BitPerFrame
	frameCount := len(inputBytes) / m.BytePerFrame

	samplePerBit := len(m.Carriers[0])

	modulatedData := make([]int32, 0, frameCount*(len(m.Preamble)+(m.BytePerFrame+1)*8*samplePerBit+m.FrameInterval))

	for i := 0; i < frameCount; i++ {
		bytes := inputBytes[i*m.BytePerFrame : (i+1)*m.BytePerFrame]

		// add the preamble
		modulatedData = append(modulatedData, m.Preamble...)

		// modulate the data
		m.CRCChecker.Reset()
		for _, b := range bytes {
			m.CRCChecker.Update(b)
			forEachBit(b, func(bit bool) {
				modulatedData = append(modulatedData, m.getCarrier(bit)...)
			})
		}
		crcByte := m.CRCChecker.Get()

		// modulate the CRC8 byte
		forEachBit(crcByte, func(bit bool) {
			modulatedData = append(modulatedData, m.getCarrier(bit)...)
		})

		fmt.Println("[Modulation] CRC8:", ByteToBool([]byte{crcByte}))

		// add the interval
		for j := 0; j < m.FrameInterval; j++ {
			modulatedData = append(modulatedData, 0)
		}

	}

	return modulatedData

}

func (m *NaiveByteModem) Demodulate(inputSignal []int32) []byte {

	samplePerBit := len(m.Carriers[0])

	type Demodulation int

	const (
		preambleDetection Demodulation = iota
		dataExtraction
	)
	state := preambleDetection

	localMaxPower := fixed.Zero
	currentWindow := make([]int32, 0, len(m.Preamble))
	frameToDecode := make([]int32, 0)
	demodulatedBytes := make([]byte, 0)

	distanceFromPotentialStart := -1

	correctionFlag := false

	type PotentialStart struct {
		power fixed.T
		index int
	}
	potentialHistory := make([]PotentialStart, 0)
	for i, currentSample := range inputSignal {

		// find the start of the signal
		if state == preambleDetection {
			if len(currentWindow) < len(m.Preamble) {
				currentWindow = append(currentWindow, currentSample)
			} else {
				currentWindow = append(currentWindow[1:], currentSample)
				power := dotProduct(currentWindow, m.Preamble)

				// find a potential start of the signal
				if power > localMaxPower && power > m.DemodulatePowerThreshold {
					fmt.Printf("[Demodulation] find a potential start of the signal at %v where power: %.2f\n", i, fixed.T(power).Float())
					localMaxPower = power
					frameToDecode = frameToDecode[:0]
					distanceFromPotentialStart = 0
					potentialHistory = append(potentialHistory, PotentialStart{power, i})
				}

				// append the currentSample to the frameToDecode if necessary
				if distanceFromPotentialStart == -1 {
					// potential start of the signal is not found yet
				} else {
					frameToDecode = append(frameToDecode, currentSample)
					distanceFromPotentialStart += 1
				}

				// a real start of the signal is found
				if distanceFromPotentialStart >= len(m.Preamble) {

					fmt.Printf("[Demodulation] find the start of the signal at %v where power: %.2f\n", i-distanceFromPotentialStart, fixed.T(localMaxPower).Float())
					fmt.Println("[Demodulation] potentialHistory:", potentialHistory)

					// determine whether to flip
					correctionFlag = false
					if len(potentialHistory) > 1 {
						lastPotentialStart := potentialHistory[len(potentialHistory)-1]
						secondLastPotentialStart := potentialHistory[len(potentialHistory)-2]
						increaseRate := fixed.T(lastPotentialStart.power - secondLastPotentialStart.power).Div(secondLastPotentialStart.power)
						deltaIndex := lastPotentialStart.index - secondLastPotentialStart.index
						fmt.Printf("[Demodulation] increaseRate: %.2f\n", fixed.T(increaseRate).Float())
						fmt.Printf("[Demodulation] deltaIndex: %d\n", deltaIndex)

						correctionFlag = increaseRate < m.CorrectionThreshold
					} else {
						fmt.Println("[Demodulation] not enough potentialHistory to determine correction, you may decrease the POWER_THRESHOLD")
					}

					fmt.Printf("[Demodulation] correctionFlag: %v\n", correctionFlag)

					localMaxPower = 0
					currentWindow = currentWindow[:0]
					distanceFromPotentialStart = -1
					state = dataExtraction
				}
			}
		}

		if state == dataExtraction {
			frameToDecode = append(frameToDecode, currentSample)

			if len(frameToDecode) == (m.BytePerFrame+1)*8*samplePerBit {

				frameBytes := make([]byte, 0, m.BytePerFrame)
				var b BitSet8
				for j := 0; j < (m.BytePerFrame+1)*8; j++ {
					s1 := dotProduct(m.Carriers[1], frameToDecode[j*samplePerBit:])
					s0 := dotProduct(m.Carriers[0], frameToDecode[j*samplePerBit:])
					if (s1 > s0) != correctionFlag {
						b.Set(7 - j%8)
					}

					if (j % 8) == 7 {
						frameBytes = append(frameBytes, b.ToByte())
					}
				}

				dataBytes := frameBytes[:m.BytePerFrame]
				crcByte := frameBytes[m.BytePerFrame]

				if !(m.CRCChecker.Check(dataBytes, crcByte)) {
					if !correctionFlag {
						fmt.Println("[Demodulation] CRC check failed before flip")
					} else {
						// Maybe we shouldn't use the correctionFlag before ?
						for i := range dataBytes {
							dataBytes[i] = ^dataBytes[i]
						}
						crcByte = ^crcByte

						if !m.CRCChecker.Check(dataBytes, crcByte) {
							fmt.Println("[Demodulation] CRC check failed after flip")
						} else {
							// Indeed, we should not use the correctionFlag before
							fmt.Println("[Demodulation] CRC check passed after flip")
						}
					}
				} else {
					fmt.Println("[Demodulation] CRC check passed")
				}
				demodulatedBytes = append(demodulatedBytes, dataBytes...)

				state = preambleDetection
				potentialHistory = potentialHistory[:0]
			}
		}
	}

	return demodulatedBytes
}

func (m *NaiveByteModem) getCarrier(bit bool) []int32 {
	if bit {
		return m.Carriers[1]
	} else {
		return m.Carriers[0]
	}
}
