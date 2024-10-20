package main

import (
	"Aethernet/cmd/project1/task3/config"
	"Aethernet/internel/utils"
)

func main() {

	track, _ := utils.ReadBinary[int32]("recorder.bin")
	outputBits := config.Modem.Demodulate(track)
	utils.WriteBinary("output.bin", outputBits)
	utils.WriteTxt("output.txt", outputBits, func(bit bool) int {
		if bit {
			return 1
		} else {
			return 0
		}
	})

}
