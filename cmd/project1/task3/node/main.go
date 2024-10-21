package main

import (
	"Aethernet/cmd/project1/task3/config"
	"Aethernet/internel/callbacks"
	"Aethernet/internel/utils"
	"fmt"

	"github.com/xsjk/go-asio"
)

func main() {

	player := callbacks.Player{}
	recorder := callbacks.Recorder{Track: make([]int32, 0, config.EXPECTED_TOTAL_BITS)}

	utils.WriteBinary("preamble.bin", config.BitModem.Preamble)

	{
		inputBits, err := utils.ReadTxt[bool]("INPUT.txt")
		if err != nil {
			panic(err)
		} else {
			fmt.Println("[Debug] Read input data from INPUT.txt", "length:", len(inputBits))
		}
		modulatedData := config.BitModem.Modulate(inputBits)
		fmt.Println("[Debug] Modulated data length:", len(modulatedData))
		// add some zero padding before sending
		player.Track = append(make([]int32, config.SAMPLE_RATE), modulatedData...)

		asio.Session{IOHandler: func(in, out [][]int32) {
			player.Update(in, out)
			recorder.Update(in, out)
		}}.Run()

		utils.WriteBinary("recorder.bin", recorder.Track)
	}

	{
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

}
