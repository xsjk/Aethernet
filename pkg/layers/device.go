package layer

import (
	"sync"
	"time"

	"github.com/xsjk/go-asio"
)

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
	SampleRate float64 // the fake sample rate, 0 means no limit
	done       chan struct{}
}

func (d *LoopbackDevice) Start(callback func([][]int32, [][]int32)) {
	d.done = make(chan struct{})
	go func() {
		var buf = make([][]int32, 2)
		buf[0] = make([]int32, 512)
		buf[1] = make([]int32, 512)

		swap := true
		update := func() {
			if swap {
				callback(buf[:1], buf[1:])
			} else {
				callback(buf[1:], buf[:1])
			}
			swap = !swap
		}

		if d.SampleRate == 0 {
			for {
				select {
				case <-d.done:
					return
				default:
					update()
				}
			}
		} else {
			ticker := time.NewTicker(time.Second / time.Duration(d.SampleRate))
			for {
				select {
				case <-d.done:
					return
				case <-ticker.C:
					update()
				}
			}
		}
	}()
}

func (d *LoopbackDevice) Stop() {
	close(d.done)
}

type CrossfeedDevice struct {
	*CrossfeedDeviceManager
	done     chan struct{}
	output   [][]int32
	input    [][]int32
	callback func([][]int32, [][]int32)
}

type CrossfeedDeviceManager struct {
	SampleRate float64 // the fake sample rate, 0 means no limit
	once       sync.Once
	done       chan struct{}
	devices    [2]*CrossfeedDevice
}

func (m *CrossfeedDeviceManager) Generate() (*CrossfeedDevice, *CrossfeedDevice) {
	buffer := make([][]int32, 2)
	buffer[0] = make([]int32, 512)
	buffer[1] = make([]int32, 512)
	m.devices[0] = &CrossfeedDevice{CrossfeedDeviceManager: m, input: buffer[:1], output: buffer[1:], done: make(chan struct{}, 2)}
	m.devices[1] = &CrossfeedDevice{CrossfeedDeviceManager: m, input: buffer[1:], output: buffer[:1], done: make(chan struct{}, 2)}
	return m.devices[0], m.devices[1]
}

func (d *CrossfeedDevice) Start(callback func([][]int32, [][]int32)) {
	if d.CrossfeedDeviceManager == nil {
		panic("CrossfeedDeviceManager is nil, use CrossfeedDeviceManager.Generate() to create a pair of devices")
	}

	d.callback = callback

	m := d.CrossfeedDeviceManager
	m.once.Do(
		func() {
			m.done = make(chan struct{})
			go func() {
				// wait for all the devices to be done
				<-m.devices[0].done
				<-m.devices[1].done
				close(m.done)
			}()
			go func() {
				if m.SampleRate == 0 {
					for {
						select {
						case <-m.done:
							return
						default:
							for _, d := range m.devices {
								if d.callback != nil {
									d.callback(d.input, d.output)
								}
							}
						}
					}
				} else {
					ticker := time.NewTicker(time.Second / time.Duration(d.SampleRate))
					for {
						select {
						case <-m.done:
							return
						case <-ticker.C:
							for _, d := range m.devices {
								if d.callback != nil {
									d.callback(d.input, d.output)
								}
							}
						}
					}
				}
			}()
		})

	d.done = make(chan struct{})

}

func (d *CrossfeedDevice) Stop() {
	d.callback = nil
	close(d.done)
}
