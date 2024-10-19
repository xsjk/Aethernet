.PHONY: run clean

bin/proj$(proj)_task$(task).exe: cmd/project$(proj)/task$(task)/main.go
	go build -o $@ $<

bin/proj$(proj)_task$(task)_node$(node).exe: cmd/project$(proj)/task$(task)/node$(node)/main.go
	go build -o $@ $<

build: bin/proj$(proj)_task$(task).exe

run: build
	cd bin && proj$(proj)_task$(task)

clean:
	del /q "bin\*.exe"
