package main

import (
	"Aethernet/cmd/project1/task3/config"
	"Aethernet/internel/callbacks"
	"Aethernet/internel/utils"

	"github.com/xsjk/go-asio"
)

func main() {

	inputBits, _ := utils.ReadTxt[bool]("input.txt")
	player := callbacks.Player{Track: config.Modem.Modulate(inputBits)}
	asio.Session{IOHandler: player.Update}.Run()

}
