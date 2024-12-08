package async

import (
	"bufio"
	"os"
	"os/signal"
	"syscall"
)

func EnterKey() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		close(done)
	}()
	return done
}

func Exit() <-chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	return sigChan
}
