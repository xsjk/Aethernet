package main

import (
	"Aethernet/cmd/project1/task3/config"
	"Aethernet/internel/callbacks"
	"Aethernet/internel/utils"

	"github.com/xsjk/go-asio"
)

func main() {

	recorder := callbacks.Recorder{Track: make([]int32, 0, config.EXPECTED_TOTAL_BITS)}
	asio.Session{IOHandler: recorder.Update}.Run()
	utils.WriteBinary("recorder.bin", recorder.Track)

	track, _ := utils.ReadBinary[int32]("recorder.bin")
	outputBits := config.BitModem.Demodulate(track)
	utils.WriteBinary("output.bin", outputBits)
	utils.WriteTxt("OUTPUT.txt", outputBits, func(bit bool) int {
		if bit {
			return 1
		} else {
			return 0
		}
	})

}
