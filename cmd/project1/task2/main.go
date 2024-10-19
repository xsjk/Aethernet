package main

import (
	"math"

	"github.com/xsjk/go-asio"
)

const sampleRate = 44100

func main() {

	offset := 0
	asio.Session{
		SampleRate: sampleRate,
		IOHandler: func(in, out [][]int32) {
			for i := range out[0] {
				t := float64(offset+i) / sampleRate
				o := math.Sin(2 * math.Pi * 1000 * t)
				o += math.Sin(2 * math.Pi * 10000 * t)
				out[0][i] = int32(o * 0x7fffffff / 2)
			}
			offset += len(out[0])
		},
	}.Run()

}
