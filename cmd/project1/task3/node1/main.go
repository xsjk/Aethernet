package main

import (
	"Aethernet/cmd/project1/task3/config"
	"Aethernet/internel/callbacks"
	"Aethernet/internel/utils"
	"fmt"

	"github.com/xsjk/go-asio"
)

func main() {

	inputBits, err := utils.ReadTxt[bool]("INPUT.txt")
	if err != nil {
		panic(err)
	} else {
		fmt.Println("[Debug] Read input data from INPUT.txt", "length:", len(inputBits))
	}
	modulatedData := config.BitModem.Modulate(inputBits)
	fmt.Println("[Debug] Modulated data length:", len(modulatedData))
	// add some zero padding before sending
	zeros := make([]int32, 10000)
	modulatedData = append(zeros, modulatedData...)
	player := callbacks.Player{Track: modulatedData}
	asio.Session{IOHandler: player.Update}.Run()

}
