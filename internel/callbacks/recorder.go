package callbacks

type Recorder struct {
	Track []int32
}

func (r *Recorder) Update(in, out [][]int32) {
	r.Track = append(r.Track, in[0]...)
}
