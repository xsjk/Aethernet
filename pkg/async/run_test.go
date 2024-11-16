package async

import (
	"testing"
	"time"
)

func TestRun(t *testing.T) {

	Run(func() []any {
		start := time.Now()
		eps := 10 * time.Millisecond
		n := 10
		args := make([]any, n)
		for i := range n {
			d := time.Duration(i) * 100 * time.Millisecond
			args[i] = func() {
				time.Sleep(d)
				duration := time.Since(start)
				if duration > d+eps || duration < d-eps {
					t.Errorf("done after %v\n", d)
				} else {
					t.Logf("done after %v\n", d)
				}
			}
		}
		return args
	}()...)

}
