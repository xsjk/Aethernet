package kernel32

import "golang.org/x/sys/windows"

var (
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	setEvent               = kernel32.NewProc("SetEvent")
	createEventW           = kernel32.NewProc("CreateEventW")
	waitForSingleObjectEx  = kernel32.NewProc("WaitForSingleObjectEx")
	waitForMultipleObjects = kernel32.NewProc("WaitForMultipleObjects")
	closeHandle            = kernel32.NewProc("CloseHandle")
)
