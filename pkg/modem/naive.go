package modem

import (
	"fmt"
	"reflect"
)

// TODO: Use FixedPoint instead of float64

type NaiveModem struct {
	Preamble []int32

	BitPerFrame   int // number of bits per frame
	FrameInterval int // interval between frames

	CRCChecker CRC8Checker

	Carriers [2][]int32

	DemodulatePowerThreshold float64
}

func (m *NaiveModem) Modulate(inputBits []bool) []int32 {

	frameCount := (len(inputBits) + m.BitPerFrame - 1) / m.BitPerFrame

	samplePerBit := len(m.Carriers[0])

	modulatedData := make([]int32, 0, frameCount*(len(m.Preamble)+(m.BitPerFrame+8)*samplePerBit+m.FrameInterval))

	for i := 0; i < frameCount; i++ {
		bits := inputBits[i*m.BitPerFrame : min((i+1)*m.BitPerFrame, len(inputBits))]

		// add the preamble
		modulatedData = append(modulatedData, m.Preamble...)

		// modulate CRC8
		crcBits := m.CRCChecker.Calculate(bits)
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

func (m *NaiveModem) Demodulate(inputSignal []int32) []bool {

	samplePerBit := len(m.Carriers[0])

	type Demodulation int

	const (
		preambleDetection Demodulation = iota
		dataExtraction
	)
	state := preambleDetection

	powerSmooth := 0.0
	localMaxPower := 0.0
	currentWindow := make([]int32, 0, len(m.Preamble))
	frameToDecode := make([]int32, 0)
	demodulatedBits := make([]bool, 0)

	distanceFromPotentialStart := -1

	correctionFlag := false

	type PotentialStart struct {
		power float64
		index int
	}
	potentialHistory := make([]PotentialStart, 0)
	for i, currentSample := range inputSignal {
		powerSmooth = powerSmooth*(1-1.0/64) + (float64(currentSample)/0x7fffffff*float64(currentSample)/0x7fffffff)/64

		// find the start of the signal
		if state == preambleDetection {
			if len(currentWindow) < len(m.Preamble) {
				currentWindow = append(currentWindow, currentSample)
			} else {
				currentWindow = append(currentWindow[1:], currentSample)
				power := dotProduct(currentWindow, m.Preamble)

				// find a potential start of the signal
				if power > localMaxPower && power > m.DemodulatePowerThreshold && power > powerSmooth {
					fmt.Printf("[Demodulation] find a potential start of the signal at %v where power: %.2f\n", i, power)
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
				if distanceFromPotentialStart >= len(m.Preamble)/2 {

					fmt.Printf("[Demodulation] find the start of the signal at %v where power: %.2f\n", i-distanceFromPotentialStart, localMaxPower)
					fmt.Println("[Demodulation] potentialHistory:", potentialHistory)

					// determine whether to flip
					correctionFlag = false
					if len(potentialHistory) > 2 {
						lastPotentialStart := potentialHistory[len(potentialHistory)-1]
						secondLastPotentialStart := potentialHistory[len(potentialHistory)-2]
						increaseRate := (lastPotentialStart.power - secondLastPotentialStart.power) / secondLastPotentialStart.power
						deltaIndex := lastPotentialStart.index - secondLastPotentialStart.index
						fmt.Printf("[Demodulation] increaseRate: %.2f\n", increaseRate)
						fmt.Printf("[Demodulation] deltaIndex: %d\n", deltaIndex)

						correctionFlag = increaseRate < 0.8
					}

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
				frameNoCarrier1 := make([]float64, len(frameToDecode))
				frameNoCarrier0 := make([]float64, len(frameToDecode))
				for j := range frameToDecode {
					frameNoCarrier1[j] = float64(frameToDecode[j]) / 0x7fffffff * float64(m.Carriers[1][j%samplePerBit]) / 0x7fffffff
					frameNoCarrier0[j] = float64(frameToDecode[j]) / 0x7fffffff * float64(m.Carriers[0][j%samplePerBit]) / 0x7fffffff
				}

				frameBits := make([]bool, 0, m.BitPerFrame+crcBitCount)
				for j := 0; j < m.BitPerFrame+crcBitCount; j++ {
					valuePerBit1 := 0.0
					valuePerBit0 := 0.0
					for k := 0; k < samplePerBit; k++ {
						valuePerBit1 += frameNoCarrier1[j*samplePerBit+k]
						valuePerBit0 += frameNoCarrier0[j*samplePerBit+k]
					}
					frameBits = append(frameBits, (valuePerBit1 > valuePerBit0) != correctionFlag)
				}

				crcBits := frameBits[:crcBitCount]
				dataBits := frameBits[crcBitCount:]
				if !reflect.DeepEqual(m.CRCChecker.Calculate(dataBits), crcBits) {
					// try flip all the bits
					for i := range dataBits {
						dataBits[i] = !dataBits[i]
					}
					for i := range crcBits {
						crcBits[i] = !crcBits[i]
					}
					if !reflect.DeepEqual(m.CRCChecker.Calculate(dataBits), crcBits) {
						fmt.Println("[Demodulation] CRC check failed after flip")
					} else {
						fmt.Println("[Demodulation] CRC check passed after flip")
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

func dotProduct(a, b []int32) float64 {
	sum := 0.0
	for i := range a {
		sum += float64(a[i]) * float64(b[i]) / 0x7fffffff / 0x7fffffff
	}
	return sum
}

func (m *NaiveModem) getCarrier(bit bool) []int32 {
	if bit {
		return m.Carriers[1]
	} else {
		return m.Carriers[0]
	}
}
