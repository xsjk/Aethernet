package main

import (
	"Aethernet/cmd/project2/task2/config"
	"Aethernet/internel/utils"
	"Aethernet/pkg/async"
	"fmt"
)

func main() {

	inputBytes, err := utils.ReadBinary[byte]("INPUT.bin")
	if err != nil {
		fmt.Println(err)
		return
	}

	layer := &config.Layer
	layer.Address = 0x01
	layer.Open()
	defer layer.Close()

	select {
	case err := <-layer.SendAsync(0x02, inputBytes):
		if err != nil {
			fmt.Println(err)
		}
	case <-async.EnterKey():
	}

}
