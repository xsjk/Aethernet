package modem

import (
	"Aethernet/pkg/fixed"
	"fmt"
	"reflect"
)

type BitModem interface {
	Modulate(inputBits []bool) []int32
	Demodulate(inputSignal []int32) []bool
}

type NaiveBitModem struct {
	Preamble []int32
	Carriers [2][]int32

	BitPerFrame   int // number of bits per frame
	FrameInterval int // interval between frames

	crcChecker CRC8Checker

	CorrectionThreshold      fixed.T
	DemodulatePowerThreshold fixed.T
}

func (m *NaiveBitModem) Modulate(inputBits []bool) []int32 {

	frameCount := (len(inputBits) + m.BitPerFrame - 1) / m.BitPerFrame

	samplePerBit := len(m.Carriers[0])

	modulatedData := make([]int32, 0, frameCount*(len(m.Preamble)+(m.BitPerFrame+8)*samplePerBit+m.FrameInterval))

	for i := 0; i < frameCount; i++ {
		bits := inputBits[i*m.BitPerFrame : min((i+1)*m.BitPerFrame, len(inputBits))]

		// add the preamble
		modulatedData = append(modulatedData, m.Preamble...)

		// modulate CRC8
		crcBits := m.crcChecker.CalculateBits(bits)
		fmt.Println("[Modulation] CRC8:", crcBits)
		for _, bit := range crcBits {
			modulatedData = append(modulatedData, m.getCarrier(bit)...)
		}

		// modulate the data
		for _, bit := range bits {
			modulatedData = append(modulatedData, m.getCarrier(bit)...)
		}
		// add the interval
		for j := 0; j < m.FrameInterval; j++ {
			modulatedData = append(modulatedData, 0)
		}

	}

	return modulatedData

}

func (m *NaiveBitModem) Demodulate(inputSignal []int32) []bool {

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
	demodulatedBits := make([]bool, 0)

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

			crcBitCount := 8

			if len(frameToDecode) == (m.BitPerFrame+crcBitCount)*samplePerBit {

				frameBits := make([]bool, 0, m.BitPerFrame+crcBitCount)
				for j := 0; j < m.BitPerFrame+crcBitCount; j++ {
					s1 := dotProduct(m.Carriers[1], frameToDecode[j*samplePerBit:])
					s0 := dotProduct(m.Carriers[0], frameToDecode[j*samplePerBit:])
					frameBits = append(frameBits, (s1 > s0) != correctionFlag)
				}

				crcBits := frameBits[:crcBitCount]
				dataBits := frameBits[crcBitCount:]
				if !reflect.DeepEqual(m.crcChecker.CalculateBits(dataBits), crcBits) {
					if !correctionFlag {
						fmt.Println("[Demodulation] CRC check failed before flip")
					} else {
						// Maybe we shouldn't use the correctionFlag before ?
						for i := range dataBits {
							dataBits[i] = !dataBits[i]
						}
						for i := range crcBits {
							crcBits[i] = !crcBits[i]
						}
						if !reflect.DeepEqual(m.crcChecker.CalculateBits(dataBits), crcBits) {
							fmt.Println("[Demodulation] CRC check failed after flip")
						} else {
							// Indeed, we should not use the correctionFlag before
							fmt.Println("[Demodulation] CRC check passed after flip")
						}
					}
				} else {
					fmt.Println("[Demodulation] CRC check passed")
				}
				demodulatedBits = append(demodulatedBits, dataBits...)

				state = preambleDetection
				potentialHistory = potentialHistory[:0]
			}
		}
	}

	return demodulatedBits
}

func (m *NaiveBitModem) getCarrier(bit bool) []int32 {
	if bit {
		return m.Carriers[1]
	} else {
		return m.Carriers[0]
	}
}
