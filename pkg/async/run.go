package async

func Run(jobs ...any) {
	Await(Gather(
		Gather(jobs...),
		EnterKey(),
	))
}
