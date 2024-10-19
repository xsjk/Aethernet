package callbacks

type Player struct {
	idx   uint
	Track []int32
}

func (p *Player) Update(in, out [][]int32) {
	out_array := out[0]
	size := min(len(out_array), len(p.Track)-int(p.idx))
	i := 0
	for i = range size {
		out_array[i] = p.Track[p.idx]
		p.idx += 1
	}
	for ; i < len(out_array); i++ {
		out_array[i] = 0
	}
}

func (p *Player) Reset() {
	p.idx = 0
}
