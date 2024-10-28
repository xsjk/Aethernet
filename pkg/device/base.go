package device

type Device interface {
	Start(callback func([]int32, []int32))
	Stop()
}

const BufferSize = 512
