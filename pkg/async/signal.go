package async

type Signal[T any] chan T

func (s *Signal[T]) Notify() {
	if *s != nil {
		select {
		case <-*s:
		default:
			close(*s)
		}
	}
}

func (s *Signal[T]) NotifyValue(value T) {
	if *s != nil {
		select {
		case <-*s:
		default:
			*s <- value
		}
	}
}

func (s *Signal[T]) Signal() <-chan T {
	*s = make(chan T)
	return *s
}

func (s *Signal[T]) Wait() T {
	return <-s.Signal()
}
