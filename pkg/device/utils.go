package device

import "golang.org/x/exp/rand"

func cleari32(a []int32) {
	for i := range a {
		a[i] = 0
	}
}

func randi32(a []int32) {
	for i := range a {
		a[i] = rand.Int31()
	}
}

func sumi32(a, b, c []int32) {
	for i := range a {
		sum := int64(a[i]) + int64(b[i])
		if sum > 0x7fffffff {
			sum = 0x7fffffff
		} else if sum < -0x80000000 {
			sum = -0x80000000
		}
		c[i] = int32(sum)
	}
}

func alloci32(n int) []int32 {
	return make([]int32, n)
}
