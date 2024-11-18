.PHONY: run clean build

ifeq ($(OS),Windows_NT)
SHELL := powershell.exe
.SHELLFLAGS := -NoProfile -Command
endif

bin/proj$(proj)_task$(task)_node$(node).exe: cmd/project$(proj)/task$(task)/node$(node)/main.go
	go build -o $@ $<

build:
	go build -o bin/proj$(proj)_task$(task)_node$(node).exe cmd/project$(proj)/task$(task)/node$(node)/main.go

run: build
	cd bin; ./proj$(proj)_task$(task)_node$(node).exe

clean:
	rm bin/*.exe
