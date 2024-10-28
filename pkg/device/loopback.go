package device

import "time"

type Loopback struct {
	SampleRate float64 // the fake sample rate, 0 means no limit
	done       chan struct{}
}

func (d *Loopback) Start(callback func([]int32, []int32)) {
	d.done = make(chan struct{})
	go func() {
		var buf = make([][]int32, 2)
		buf[0] = alloci32(BufferSize)
		buf[1] = alloci32(BufferSize)

		swap := true
		update := func() {
			if swap {
				callback(buf[0], buf[1])
			} else {
				callback(buf[1], buf[0])
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

func (d *Loopback) Stop() {
	close(d.done)
}
