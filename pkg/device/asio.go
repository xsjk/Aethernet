package device

import "github.com/xsjk/go-asio"

type ASIOMono struct {
	DeviceName string
	SampleRate float64
	InChannel  int
	OutChannel int
	device     asio.Device
}

func (a *ASIOMono) Start(callback func([]int32, []int32)) {
	a.device.Load(a.DeviceName)
	a.device.SetSampleRate(a.SampleRate)
	a.device.Open()
	a.device.Start(func(in, out [][]int32) {
		callback(in[a.InChannel], out[a.OutChannel])
	})
}

func (a *ASIOMono) Stop() {
	a.device.Stop()
	a.device.Close()
	a.device.Unload()
}
