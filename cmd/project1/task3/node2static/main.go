package main

import (
	"Aethernet/cmd/project1/task3/config"
	"Aethernet/internel/utils"
)

func main() {

	track, err := utils.ReadBinary[int32]("recorder.bin")
	if err != nil {
		panic(err)
	}
	println("[Debug] Read recorded data from recorder.bin", "length:", len(track))
	outputBits := config.Modem.Demodulate(track)
	println("[Debug] Demodulated data length:", len(outputBits))
	utils.WriteBinary("output.bin", outputBits)
	utils.WriteTxt("OUTPUT.txt", outputBits, func(bit bool) int {
		if bit {
			return 1
		} else {
			return 0
		}
	})

}
