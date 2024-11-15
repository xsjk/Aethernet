package main

import (
	"Aethernet/cmd/project2/task2/config"
	"Aethernet/internel/utils"
)

func main() {

	inputBytes, _ := utils.ReadBinary[byte]("INPUT.bin")

	layer := &config.Layer
	layer.Address = 0x01
	layer.Open()
	layer.Send(0x02, inputBytes)
	layer.Close()

}
