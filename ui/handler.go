package ui

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	messageBoxW = user32.NewProc("MessageBoxW")
	IDIGNORE    = 5
	IDRETRY     = 4
)

func PortForwardNotPossible() (result int) {
	text := fmt.Sprintf(
		"Could not automatically forward Ports.\n\n" +
			"Please enable UPnP in your routers interface.\n\n" +
			"Or forward the following ports manually:\n")

	caption := "VRC-Haven"

	ret, _, _ := messageBoxW.Call(0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(caption))),
		2)

	result = int(ret)
	return
}
