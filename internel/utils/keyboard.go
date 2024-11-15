package utils

import (
	"bufio"
	"os"
)

func WaitEnterAsync() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		close(done)
	}()
	return done
}
