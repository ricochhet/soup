package quickedit

import (
	"syscall"
	"unsafe"
)

const (
	EnableQuickEditMode = 0x0040
	EnableExtendedFlags = 0x0080
)

func QuickEdit(v bool) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	handle, err := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	if err != nil {
		return err
	}

	var mode uint32

	r1, _, err := getConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	if r1 == 0 {
		return err
	}

	mode |= EnableExtendedFlags

	if v {
		mode |= EnableQuickEditMode
	} else {
		mode &^= EnableQuickEditMode
	}

	r1, _, err = setConsoleMode.Call(uintptr(handle), uintptr(mode))
	if r1 == 0 {
		return err
	}

	return nil
}
