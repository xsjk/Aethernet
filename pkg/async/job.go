package async

func Job(f func()) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()
	return done
}
