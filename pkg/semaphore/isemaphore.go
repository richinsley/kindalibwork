package semaphore

type ISemaphore interface {
	// increase the Semaphore count by one
	// return 0 if successfull
	Post() int

	// wait for semaphore to be non-zero
	// decrease the Semaphore count by one
	// return 0 if successfull
	Wait() int

	// wait for n msecs, return 0 if success, non-zero for timeout/error
	WaitTimed(int) int

	// Try to wait on a semaphore.  If semaphore count is non-zero, semaphore count is decreased,
	// and returns 0.  If semaphore count is zero, returns 1
	TryWait() int

	// reset a sempahore's count to 0
	Reset() int

	// get the native semaphore handle
	GetHandle() uintptr

	// close the semaphore
	Close() int
}
