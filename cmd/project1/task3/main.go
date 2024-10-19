package main

import (
	"Aethernet/internel/utils"
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/xsjk/go-asio"
)

const (
	PREAMBLE_LENGTH     = 10000
	PREAMBLE_START_FREQ = 6000.0
	PREAMBLE_END_FREQ   = 12000.0
	SAMPLE_RATE         = 48000.0

	SAMPLE_PER_BIT      = 30
	EXPECTED_TOTAL_BITS = 1000
	BIT_PER_FRAME       = 1000

	AMPLITUDE  = 1.0
	ONE_FREQ   = 800
	ZERO_FREQ  = 1400
	ONE_PHASE  = 0
	ZERO_PHASE = math.Pi

	POWER_THRESHOLD = 50
	CRC_BITS        = utils.CRC_BITS
)

var preamble []float64 = utils.GeneratePreamble(
	PREAMBLE_START_FREQ,
	PREAMBLE_END_FREQ,
	PREAMBLE_LENGTH,
	SAMPLE_RATE,
)

func Modulation(modulatedData chan []int32, inputBits []bool) {

	modulationDebug := make([]float64, 0)

	frameCount := (len(inputBits) + BIT_PER_FRAME - 1) / BIT_PER_FRAME

	oneSignal := make([]float64, SAMPLE_PER_BIT)
	zeroSignal := make([]float64, SAMPLE_PER_BIT)

	// get the signal for one and zero
	for i := 0; i < SAMPLE_PER_BIT; i++ {
		t := float64(i) / SAMPLE_RATE
		oneSignal[i] = AMPLITUDE * math.Sin(2*math.Pi*ONE_FREQ*t+ONE_PHASE)
		zeroSignal[i] = AMPLITUDE * math.Sin(2*math.Pi*ZERO_FREQ*t+ZERO_PHASE)
	}

	// // reserve the space for modulatedData if needed
	// println("frameCount:", frameCount)
	// println("len(inputBits):", len(inputBits))

	// for i := 0; i < 100; i += 5 {
	// 	fmt.Println(inputBits[i : i+5])
	// }

	for i := 0; i < frameCount; i++ {
		end := min((i+1)*BIT_PER_FRAME, len(inputBits))
		frameBits := inputBits[i*BIT_PER_FRAME : end]
		// fmt.Printf("[Debug] Modulate chunk %v\n", frameBits)

		frameData := make([]float64, 0, len(preamble)+len(frameBits)*SAMPLE_PER_BIT)

		// Preamble
		frameData = append(frameData, preamble...)

		// Bits
		crcBits := utils.CalCRC8(frameBits)

		// modulate the data (inputdata + CRC8)
		for _, bit := range frameBits {
			var currentSignal []float64
			if bit {
				currentSignal = oneSignal
			} else {
				currentSignal = zeroSignal
			}
			frameData = append(frameData, currentSignal...)
		}

		// modulate the data (inputdata + CRC8)
		for _, bit := range crcBits {
			var currentSignal []float64
			if bit {
				currentSignal = oneSignal
			} else {
				currentSignal = zeroSignal
			}
			frameData = append(frameData, currentSignal...)
		}

		// fmt.Printf("[Modulation] nbit: %d nsamples: %d\n", len(frameData), len(frameData))

		// add the frame to the
		frameDataInt32 := make([]int32, 0, len(frameData))
		for _, sample := range frameData {
			frameDataInt32 = append(frameDataInt32, int32(math.Max(math.Min(sample, 1), -1)*math.MaxInt32))
			modulationDebug = append(modulationDebug, sample)
		}
		modulatedData <- frameDataInt32
	}

	utils.WriteBinary("modulationDebug.bin", modulationDebug)
}

// put the bit packets into the BitsBuffer
type Modulator struct {
	OutputBuffer chan []int32
	InputBuffer  chan []bool
}

func (m *Modulator) Mainloop() {
	for bits := range m.InputBuffer {
		Modulation(m.OutputBuffer, bits)
	}
}

// Consumer is a struct that consumes the buffer and send to output
type Consumer struct {
	Buffer  chan []int32
	current []int32
}

var outDebug *os.File

func (p *Consumer) Update(in, out [][]int32) {

	out_array := out[0]
	i := 0
	for i < len(out_array) {
		// fmt.Printf("[Consumer] i: %d\n", i)
		if p.current == nil {
			select {
			case p.current = <-p.Buffer:
			default:
				out_array[i] = 0
				i += 1
			}
		} else {
			// fmt.Printf("[Consumer] len(p.current): %d\n", len(p.current))

			n := copy(out_array[i:], p.current)
			i += n
			p.current = p.current[n:]

			if len(p.current) == 0 {
				p.current = nil
				// fmt.Printf("[Consumer] p.current is nil\n")
			}

		}
	}

	if outDebug == nil {
		outDebug, _ = os.Create("outDebug.bin")
	}
	// write the output to the file
	binary.Write(outDebug, binary.LittleEndian, out_array)
}

// Producer is a struct that reads the input and put into the buffer
type Producer struct {
	Buffer    chan []int32
	pool      *sync.Pool
	debugFile *os.File
}

func (p *Producer) Update(in, out [][]int32) {
	in_array := in[0]

	if p.pool == nil {
		p.pool = &sync.Pool{
			New: func() interface{} {
				chunk := make([]int32, len(in_array))
				return &chunk
			},
		}
	}

	if p.debugFile == nil {
		p.debugFile, _ = os.Create("inDebug.bin")
	}
	binary.Write(p.debugFile, binary.LittleEndian, in_array)

	chunk := *p.pool.Get().(*[]int32)
	copy(chunk, in_array)
	select {
	case p.Buffer <- chunk:
	default:
		fmt.Println("[Producer] Buffer is full, consider increasing the buffer size")
	}

}

// return the chunk to the pool
func (p *Producer) Return(chunk *[]int32) {
	p.pool.Put(chunk)
}

// put the audio packets into the AudioBuffer
type Demodulator struct {
	InputBuffer  chan []int32
	OutputBuffer chan []bool
}

func (d *Demodulator) Mainloop() {
	for {
		Demodulation(d.InputBuffer, d.OutputBuffer)
	}
}

func Demodulation(receivingData chan []int32, demodulatedBits chan []bool) {

	oneSignal := make([]float64, SAMPLE_PER_BIT)
	zeroSignal := make([]float64, SAMPLE_PER_BIT)

	// get the signal for one
	for i := 0; i < SAMPLE_PER_BIT; i++ {
		t := float64(i) / SAMPLE_RATE
		oneSignal[i] = AMPLITUDE * math.Sin(2*math.Pi*ONE_FREQ*t+ONE_PHASE)
		zeroSignal[i] = AMPLITUDE * math.Sin(2*math.Pi*ZERO_FREQ*t+ZERO_PHASE)
	}
	// get the dot product value of oneSignal

	// erase 0 at the beginning
	// firstNonZeroIndex := 0
	// for firstNonZeroIndex < len(receivingData) && receivingData[firstNonZeroIndex] == 0.0 {
	// 	firstNonZeroIndex++
	// }
	// receivingData = receivingData[firstNonZeroIndex:]

	type Demodulation int

	const (
		preambleDetection Demodulation = iota
		dataExtraction
	)
	state := preambleDetection

	powerSmooth := 0.0
	localMaxPower := 0.0
	currentWindow := make([]float64, 0, PREAMBLE_LENGTH)
	frameToDecode := make([]float64, 0)
	powerDebug := make([]float64, 0)

	powerDebugFile, _ := os.Create("powerDebug.bin")
	defer powerDebugFile.Close()

	correctionFlag := false
	i := 0
	distanceFromPotentialStart := -1

	type PotentialStart struct {
		power float64
		index int
	}
	potentialHistory := make([]PotentialStart, 0)
	for currentChunk := range receivingData {
		for _, currentSample := range currentChunk {
			currentSample := float64(currentSample) / math.MaxInt32

			powerSmooth = powerSmooth*(1-1.0/64) + (currentSample*currentSample)/64

			// find the start of the signal
			if state == preambleDetection {
				if len(currentWindow) < PREAMBLE_LENGTH {
					currentWindow = append(currentWindow, currentSample)
					// fmt.Printf("currentWindow length: %d\n", len(currentWindow))

					binary.Write(powerDebugFile, binary.LittleEndian, powerDebug)

				} else {

					// find a real start of the signal and switch to receive the frame
					// a real start of the signal is found if the potential start of the signal was found and
					// no new potential start of the signal can be found within PREAMBLE_LENGTH / 2 samples
					if distanceFromPotentialStart < PREAMBLE_LENGTH/2 {

						currentWindow = append(currentWindow[1:], currentSample)
						power := dotProduct(currentWindow, preamble)

						binary.Write(powerDebugFile, binary.LittleEndian, power)

						// find a potential start of the signal
						if power > localMaxPower && power > POWER_THRESHOLD && power > powerSmooth {

							fmt.Printf("[Demodulation] find a potential start of the signal at %v where power: %.2f\n", i, power)
							// a potential start of the signal is found
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

					} else {
						// a real start of the signal is found
						fmt.Printf("[Demodulation] find the start of the signal at %v where power: %.2f\n", i-distanceFromPotentialStart, localMaxPower)
						fmt.Println("[Demodulation] potentialHistory:", potentialHistory)

						// determine whether to flip
						if len(potentialHistory) > 2 {
							lastPotentialStart := potentialHistory[len(potentialHistory)-1]
							secondLastPotentialStart := potentialHistory[len(potentialHistory)-2]
							increaseRate := (lastPotentialStart.power - secondLastPotentialStart.power) / secondLastPotentialStart.power
							deltaIndex := lastPotentialStart.index - secondLastPotentialStart.index
							fmt.Printf("[Demodulation] increaseRate: %.2f\n", increaseRate)
							fmt.Printf("[Demodulation] deltaIndex: %d\n", deltaIndex)
							correctionFlag = increaseRate < 0.5
						} else {
							correctionFlag = false
						}

						localMaxPower = 0

						// clear currentWindow
						currentWindow = currentWindow[:0]
						distanceFromPotentialStart = -1
						// switch to state 1
						state = dataExtraction

					}
				}
			}

			if state == dataExtraction {
				frameToDecode = append(frameToDecode, currentSample)
				if len(frameToDecode) == (BIT_PER_FRAME+CRC_BITS)*SAMPLE_PER_BIT {

					frameNoCarrier1 := make([]float64, len(frameToDecode))
					frameNoCarrier0 := make([]float64, len(frameToDecode))
					utils.WriteBinary("frameToDecode.bin", frameToDecode)
					for j := range frameToDecode {
						frameNoCarrier1[j] = frameToDecode[j] * oneSignal[j%SAMPLE_PER_BIT]
						frameNoCarrier0[j] = frameToDecode[j] * zeroSignal[j%SAMPLE_PER_BIT]
					}

					frameBits := make([]bool, 0, BIT_PER_FRAME+CRC_BITS)
					for j := 0; j < BIT_PER_FRAME+CRC_BITS; j++ {
						valuePerBit1 := 0.0
						valuePerBit0 := 0.0
						for k := 0; k < SAMPLE_PER_BIT; k++ {
							valuePerBit1 += frameNoCarrier1[j*SAMPLE_PER_BIT+k]
							valuePerBit0 += frameNoCarrier0[j*SAMPLE_PER_BIT+k]
						}
						frameBits = append(frameBits, (valuePerBit1 > valuePerBit0) != correctionFlag)
					}

					utils.WriteBinary("frameBits.bin", frameBits)

					dataBits := frameBits[:BIT_PER_FRAME]

					crcBits := frameBits[BIT_PER_FRAME:]
					crc := utils.CalCRC8(dataBits)
					if reflect.DeepEqual(crc, crcBits) {

						// send the bitData to the demodulatedBits
						select {
						case demodulatedBits <- dataBits:
						default:
							fmt.Println("[Demodulation] DemodulatedBits input channel is full, consider increasing the buffer size")
						}

					} else {

						fmt.Println("[Demodulation] CRC8 check failed")
					}

					// frameToDecode = frameToDecode[:0]
					state = preambleDetection
					potentialHistory = potentialHistory[:0]

				}
			}
			i += 1
		}
	}
}

func dotProduct(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	result := 0.0
	for i := range a {
		result += a[i] * b[i]
	}
	return result
}

func main() {

	utils.WriteBinary("preamble.bin", preamble)

	// print cwd
	cwd, _ := os.Getwd()
	fmt.Println("Current working directory:", cwd)

	modulator := Modulator{
		OutputBuffer: make(chan []int32, 1000),
		InputBuffer:  make(chan []bool, 1000),
	}

	demodulator := Demodulator{
		InputBuffer:  make(chan []int32, 1000),
		OutputBuffer: make(chan []bool, 1000),
	}

	// Start the modulator
	go modulator.Mainloop()
	// Start the demodulator
	go demodulator.Mainloop()

	// Print the demodulated data
	go func() {

		file, _ := os.Create("demodulatedData.bin")
		defer file.Close()
		defer fmt.Println("[Debug] Demodulated data saved to demodulatedData.bin")

		i := 0
		for bits := range demodulator.OutputBuffer {
			binary.Write(file, binary.LittleEndian, bits)
			i += len(bits)
			fmt.Println("[Debug] Demodulated data length:", i)
			// fmt.Print("[Debug] Demodulated data length:", len(bits), " total length: ", i, "\n")
			// fmt.Println("[Debug] Demodulated data length:", len(bits))
			// fmt.Print("[Debug] Demodulated data: ")
			// for _, bit := range bits {
			// 	write the bit to the file
			// 	fmt.Print(boolToInt(bit), " ")
			// }
			// fmt.Println()
			if i == EXPECTED_TOTAL_BITS {
				break
			}
		}
	}()

	use_file_input := true

	if use_file_input {
		// Manually pass the data to the modulator
		go func() {
			input, ok := utils.ReadBinary[bool]("input.bin")

			if ok != nil {
				fmt.Println(ok)
				return
			}

			time.Sleep(1 * time.Second)
			modulator.InputBuffer <- input
			close(modulator.InputBuffer)
		}()
	}

	use_real_audio := true

	if use_real_audio {
		// Use audio device to play the modulated data and record to the input of the demodulator

		consumer := Consumer{
			Buffer: modulator.OutputBuffer,
		}

		producer := Producer{
			Buffer: demodulator.InputBuffer,
		}

		recordFile, _ := os.Create("record.bin")
		defer recordFile.Close()

		asio.Session{
			SampleRate: SAMPLE_RATE,
			IOHandler: func(in, out [][]int32) {
				binary.Write(recordFile, binary.LittleEndian, in[0])
				consumer.Update(in, out)
				producer.Update(in, out)
			},
		}.Run()
	} else {

		for data := range modulator.OutputBuffer {
			demodulator.InputBuffer <- data
		}
		close(demodulator.InputBuffer)

		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}

	// utils.WriteBinary("preamble.bin", preamble)

}
