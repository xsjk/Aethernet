package callbacks

type Recoder struct {
	Track []int32
}

func (r *Recoder) Update(in, out [][]int32) {
	r.Track = append(r.Track, in[0]...)
}
