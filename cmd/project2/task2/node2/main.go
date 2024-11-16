package main

import (
	"Aethernet/cmd/project2/task2/config"
	"Aethernet/internel/utils"
	"Aethernet/pkg/async"
	"fmt"
)

func main() {

	layer := &config.Layer
	layer.Address = 0x02
	layer.Open()
	defer layer.Close()

	select {
	case outputBytes := <-layer.ReceiveAsync():
		fmt.Printf("Received %d bytes\n", len(outputBytes))
		utils.WriteBinary("OUTPUT.bin", outputBytes)
	case <-async.EnterKey():
	}

}
