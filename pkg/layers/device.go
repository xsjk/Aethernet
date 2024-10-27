package layer

import "github.com/xsjk/go-asio"

type Device interface {
	Start(callback func([][]int32, [][]int32))
	Stop()
}

type ASIODevice struct {
	DeviceName string
	SampleRate float64
	device     asio.Device
}

func (a *ASIODevice) Start(callback func([][]int32, [][]int32)) {
	a.device.Load(a.DeviceName)
	a.device.SetSampleRate(a.SampleRate)
	a.device.Open()
	a.device.Start(callback)
}

func (a *ASIODevice) Stop() {
	a.device.Stop()
	a.device.Close()
	a.device.Unload()
}

type LoopbackDevice struct {
	done chan struct{}
}

func (d *LoopbackDevice) Start(callback func([][]int32, [][]int32)) {
	d.done = make(chan struct{})
	go func() {
		var buf = make([][]int32, 2)
		buf[0] = make([]int32, 512)
		buf[1] = make([]int32, 512)

		swap := true
		for {

			select {
			case <-d.done:
				return
			default:
				if swap {
					callback(buf[:1], buf[1:])
				} else {
					callback(buf[1:], buf[:1])
				}
				swap = !swap
			}
		}
	}()
}

func (d *LoopbackDevice) Stop() {
	close(d.done)
}
