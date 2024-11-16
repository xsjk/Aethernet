package async

func Await0(a <-chan struct{}) {
	<-a
}

func Await[R any](a <-chan R) R {
	return <-a
}

func Await2[R1 any, R2 any](a <-chan struct {
	R1 R1
	R2 R2
}) (R1, R2) {
	r := <-a
	return r.R1, r.R2
}

func Await3[R1 any, R2 any, R3 any](a <-chan struct {
	R1 R1
	R2 R2
	R3 R3
}) (R1, R2, R3) {
	r := <-a
	return r.R1, r.R2, r.R3
}

func Await4[R1 any, R2 any, R3 any, R4 any](a <-chan struct {
	R1 R1
	R2 R2
	R3 R3
	R4 R4
}) (R1, R2, R3, R4) {
	r := <-a
	return r.R1, r.R2, r.R3, r.R4
}
