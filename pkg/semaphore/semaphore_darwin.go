//go:build darwin
// +build darwin

package semaphore

/*
	#include <stdlib.h>
    #include <mach/mach.h>
    #include <mach/semaphore.h>
    #include <mach/task.h>
    #include <device/device_port.h>
    #include <pthread.h>
    #include <mach/clock.h>
	#include <mach/mach_time.h>

	semaphore_t * create_semaphore() {
		semaphore_t * m = (semaphore_t *)malloc(sizeof(semaphore_t));

		mach_port_t self = mach_task_self();
		semaphore_create(self, m, SYNC_POLICY_FIFO, 0);
		return m;
	}

	int post_semaphore(semaphore_t * m) {
		return (int)semaphore_signal(*m);
	}

	int wait_semaphore(semaphore_t * m) {
		return (int)semaphore_wait(*m);
	}

	int wait_timed_semaphore(semaphore_t * m, int msecs) {
		mach_timespec_t ts;
		ts.tv_sec = msecs / 1000;
		ts.tv_nsec = (msecs % 1000) * 1000000;
		return (int)semaphore_timedwait(*m, ts);
	}

	int try_wait_semaphore(semaphore_t * m) {
		mach_timespec_t ts;
    	ts.tv_sec = 0;
		ts.tv_nsec = 0;
		return (int)semaphore_timedwait(*m, ts);
	}

	int reset_semaphore(semaphore_t * m) {
		while(!try_wait_semaphore(m)) {}
		return 0;
	}
*/
import "C"
import "unsafe"

type Semaphore struct {
	semhandle uintptr
}

func NewSemaphore() *Semaphore {
	retv := &Semaphore{
		// uintptr(unsafe.Pointer(value))
		semhandle: uintptr(unsafe.Pointer(C.create_semaphore())),
	}
	return retv
}

func (s *Semaphore) Post() int {
	return int(C.post_semaphore((*C.semaphore_t)(unsafe.Pointer(s.semhandle))))
}

func (s *Semaphore) Wait() int {
	return int(C.wait_semaphore((*C.semaphore_t)(unsafe.Pointer(s.semhandle))))
}

func (s *Semaphore) WaitTimed(msecs int) int {
	return int(C.wait_timed_semaphore((*C.semaphore_t)(unsafe.Pointer(s.semhandle)), C.int(msecs)))
}

func (s *Semaphore) TryWait() int {
	return int(C.try_wait_semaphore((*C.semaphore_t)(unsafe.Pointer(s.semhandle))))
}

func (s *Semaphore) Reset() int {
	return int(C.reset_semaphore((*C.semaphore_t)(unsafe.Pointer(s.semhandle))))
}

func (s *Semaphore) GetHandle() uintptr {
	return s.semhandle
}

func (s *Semaphore) Close() int {
	return 0
}
