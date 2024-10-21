package layer

type Layer[T any, U any] interface {
	UpwardLoop(in chan T, out chan U)
	DownwardLoop(in chan U, out chan T)
}
