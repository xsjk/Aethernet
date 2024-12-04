package layers

type NaiveDataLinkLayer struct {
	PhysicalLayer

	Address    byte
	BufferSize int

	outChan chan []byte
}

func (l *NaiveDataLinkLayer) Open() {
	l.PhysicalLayer.Open()
	l.outChan = make(chan []byte, l.BufferSize)
	go func() {
		for data := range l.PhysicalLayer.ReceiveAsync() {
			if data[0] != l.Address {
				// the packet was sent by someone else
				l.outChan <- data[1:]
			}
		}
	}()
}

func (l *NaiveDataLinkLayer) SendAsync(data []byte) <-chan bool {
	return l.PhysicalLayer.SendAsync(append([]byte{l.Address}, data...))
}

func (l *NaiveDataLinkLayer) Send(data []byte) {
	<-l.SendAsync(data)
}

func (l *NaiveDataLinkLayer) ReceiveAsync() <-chan []byte {
	return l.outChan
}

func (l *NaiveDataLinkLayer) Receive() []byte {
	return <-l.ReceiveAsync()
}
