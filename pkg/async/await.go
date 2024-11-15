package async

func Await(f func()) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()
	return done
}

func AwaitResult[R any](f func() R) <-chan R {
	out := make(chan R)
	go func() {
		out <- f()
	}()
	return out
}
