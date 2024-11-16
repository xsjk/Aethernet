package async

import (
	"bufio"
	"os"
)

func EnterKey() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		close(done)
	}()
	return done
}
