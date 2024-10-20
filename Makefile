.PHONY: run clean

bin/proj$(proj)_task$(task)_node$(node).exe: cmd/project$(proj)/task$(task)/node$(node)/main.go
	go build -o $@ $<

build: bin/proj$(proj)_task$(task)_node$(node).exe

run: build
	cd bin && proj$(proj)_task$(task)_node$(node).exe

clean:
	del /q "bin\*.exe"
