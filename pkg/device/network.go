package device

import (
	"sync"
	"time"
)

type NetworkConfig[BufferIDType comparable] []struct {
	In  BufferIDType
	Out BufferIDType
}

type networkNode[BufferIDType comparable] struct {
	*Network[BufferIDType]
	done     chan struct{}
	input    []int32
	output   []int32
	callback func([]int32, []int32)
}

type Network[BufferIDType comparable] struct {
	SampleRate float64                     // the fake sample rate, 0 means no limit
	Config     NetworkConfig[BufferIDType] // the topology of the network
	LateUpdate func()                      // the post process function

	once    sync.Once
	buffers map[BufferIDType][]int32
	devices []*networkNode[BufferIDType]
	done    chan struct{}
}

func (n *Network[BufferIDType]) Stop() {
	for _, d := range n.devices {
		d.callback = nil
	}
	close(n.done)
}

func (n *Network[BufferIDType]) Join() {
	<-n.done
}

func (n *Network[BufferIDType]) GetBuffer(name BufferIDType) []int32 {
	buf, ok := n.buffers[name]
	if !ok {
		buf = alloci32(BufferSize)
		n.buffers[name] = buf
	}
	return buf
}

func (n *Network[BufferIDType]) Build() []*networkNode[BufferIDType] {
	n.buffers = make(map[BufferIDType][]int32)
	n.done = make(chan struct{})
	for _, deviceConfig := range n.Config {
		n.devices = append(n.devices, &networkNode[BufferIDType]{
			Network: n,
			input:   n.GetBuffer(deviceConfig.In),
			output:  alloci32(BufferSize),
		})
	}
	return n.devices
}

func (n *Network[BufferIDType]) update() {

	for _, d := range n.devices {
		if d.callback != nil {
			d.callback(d.input, d.output)
		}
	}

	// clear the buffers
	for _, buf := range n.buffers {
		cleari32(buf)
	}

	// sum up the output of all the devices to the input buffer
	for i, deviceConfig := range n.Config {
		device := n.devices[i]
		buf := n.buffers[deviceConfig.Out]
		sumi32(buf, device.output, buf)
	}

	if n.LateUpdate != nil {
		n.LateUpdate()
	}

}

func (d *networkNode[BufferIDType]) Start(callback func([]int32, []int32)) {

	d.callback = callback

	n := d.Network
	n.once.Do(
		func() {
			n.done = make(chan struct{})
			go func() {
				// wait for all the devices to be done
				for _, d := range n.devices {
					<-d.done
				}
				close(n.done)
			}()
			go func() {

				if n.SampleRate == 0 {
					for {
						select {
						case <-n.done:
							return
						default:
							n.update()
						}
					}
				} else {
					ticker := time.NewTicker(time.Second / time.Duration(n.SampleRate))
					for {
						select {
						case <-n.done:
							return
						case <-ticker.C:
							n.update()
						}
					}
				}
			}()
		},
	)

	d.done = make(chan struct{})
}

func (d *networkNode[BufferIDType]) Stop() {
	d.callback = nil
	close(d.done)
}
