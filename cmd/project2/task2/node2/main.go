package main

import (
	"Aethernet/cmd/project2/task2/config"
	"Aethernet/internel/utils"
)

func main() {

	layer := &config.Layer
	layer.Address = 0x02
	layer.Open()
	outputBytes := layer.Receive()
	layer.Close()

	utils.WriteBinary("OUTPUT.bin", outputBytes)

}
