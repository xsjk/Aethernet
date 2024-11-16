package async

func Promise[R any](f func() R) <-chan R {
	out := make(chan R)
	go func() {
		out <- f()
	}()
	return out
}
