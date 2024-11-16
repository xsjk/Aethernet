package async

import (
	"reflect"
)

func Gather0(c ...<-chan struct{}) <-chan struct{} {
	return Job(func() {
		for _, f := range c {
			<-f
		}
	})
}

func GatherN[R any](cs ...<-chan R) <-chan []R {
	return Promise(func() []R {
		results := make([]R, len(cs))
		for i, f := range cs {
			results[i] = <-f
		}
		return results
	})
}

func Gather(chans ...any) <-chan []any {

	for i, ch := range chans {
		v := reflect.ValueOf(ch)

		// wrap function in a promise
		if v.Kind() == reflect.Func && v.Type().NumIn() == 0 {
			f := v
			if v.Type().NumOut() == 0 {
				chans[i] = Promise(func() any {
					f.Call(nil)
					return nil
				})
			} else if v.Type().NumOut() == 1 {
				chans[i] = Promise(func() any {
					return f.Call(nil)[0].Interface()
				})
			} else {
				panic("function must return a single value or no value")
			}

		} else if v.Kind() != reflect.Chan || v.Type().ChanDir()&reflect.RecvDir == 0 {
			panic("argument must be a receive channel or a function")
		}
	}

	return Promise(func() []any {
		results := make([]any, len(chans))

		for i, ch := range chans {
			v := reflect.ValueOf(ch)
			val, ok := v.Recv()
			if !ok {
				continue
			}
			results[i] = val.Interface()
		}
		return results
	})
}

func Gather2[R1 any, R2 any](c1 <-chan R1, c2 <-chan R2) <-chan struct {
	R1 R1
	R2 R2
} {
	return Promise(func() struct {
		R1 R1
		R2 R2
	} {
		return struct {
			R1 R1
			R2 R2
		}{<-c1, <-c2}
	})
}

func Gather3[R1 any, R2 any, R3 any](c1 <-chan R1, c2 <-chan R2, c3 <-chan R3) <-chan struct {
	R1 R1
	R2 R2
	R3 R3
} {
	return Promise(func() struct {
		R1 R1
		R2 R2
		R3 R3
	} {
		return struct {
			R1 R1
			R2 R2
			R3 R3
		}{<-c1, <-c2, <-c3}
	})

}

func Gather4[R1 any, R2 any, R3 any, R4 any](c1 <-chan R1, c2 <-chan R2, c3 <-chan R3, c4 <-chan R4) <-chan struct {
	R1 R1
	R2 R2
	R3 R3
	R4 R4
} {
	return Promise(func() struct {
		R1 R1
		R2 R2
		R3 R3
		R4 R4
	} {
		return struct {
			R1 R1
			R2 R2
			R3 R3
			R4 R4
		}{<-c1, <-c2, <-c3, <-c4}
	})
}
