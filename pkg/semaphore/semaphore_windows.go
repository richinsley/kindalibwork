//go:build windows
// +build windows

package semaphore

import (
	"syscall"
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procCreateSemaphore     = kernel32.NewProc("CreateSemaphoreW")
	procReleaseSemaphore    = kernel32.NewProc("ReleaseSemaphore")
	procWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	procCloseHandle         = kernel32.NewProc("CloseHandle")
)

const (
	WAIT_OBJECT_0 = 0
	WAIT_TIMEOUT  = 0x00000102
	WAIT_FAILED   = 0xFFFFFFFF
	INFINITE      = 0xFFFFFFFF
)

type Semaphore struct {
	handle syscall.Handle
}

func NewSemaphore() *Semaphore {
	handle, _, _ := procCreateSemaphore.Call(0, uintptr(0), 0x7FFFFFFF, 0)
	if handle == 0 {
		return nil
	}
	return &Semaphore{handle: syscall.Handle(handle)}
}

func (s *Semaphore) Post() bool {
	ret, _, _ := procReleaseSemaphore.Call(uintptr(s.handle), 1, 0)
	return ret != 0
}

func (s *Semaphore) Wait() int {
	ret, _, _ := procWaitForSingleObject.Call(uintptr(s.handle), INFINITE)
	return int(ret)
}

func (s *Semaphore) WaitTimed(msecs int) int {
	ret, _, _ := procWaitForSingleObject.Call(uintptr(s.handle), uintptr(msecs))
	return int(ret)
}

func (s *Semaphore) TryWait() int {
	ret, _, _ := procWaitForSingleObject.Call(uintptr(s.handle), 0)
	return int(ret)
}

func (s *Semaphore) Reset() {
	for s.TryWait() == WAIT_OBJECT_0 {
	}
}

func (s *Semaphore) GetHandle() uintptr {
	return uintptr(s.handle)
}

func (s *Semaphore) Close() {
	procCloseHandle.Call(uintptr(s.handle))
}
