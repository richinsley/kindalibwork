//go:build linux
// +build linux

package semaphore

/*
    #include <time.h>
    #include <semaphore.h>
    #include <pthread.h>
    #include <errno.h>

	sem_t * create_semaphore() {
		sem_t * m = (sem_t *)malloc(sizeof(sem_t));
		sem_init(m, 0, 0);
		return m;
	}

	int post_semaphore(sem_t * m) {
		return (int)sem_post(m);
	}

	int wait_semaphore(sem_t * m) {
		return (int)sem_wait(m);
	}

	int wait_timed_semaphore(sem_t * m, int msecs) {
		struct timespec ts;
		clock_gettime(CLOCK_REALTIME, &ts);
		ts.tv_sec += msecs / 1000;
		ts.tv_nsec += (msecs % 1000) * 1000000;
		int retv = (int)sem_timedwait(m, &ts);
		if(retv)
		{
			retv = errno;
		}
		return retv;
	}

	int try_wait_semaphore(sem_t * m) {
		return (int)sem_trywait(m);
	}

	int reset_semaphore(sem_t * m) {
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
	return int(C.post_semaphore((*C.sem_t)(unsafe.Pointer(s.semhandle))))
}

func (s *Semaphore) Wait() int {
	return int(C.wait_semaphore((*C.sem_t)(unsafe.Pointer(s.semhandle))))
}

func (s *Semaphore) WaitTimed(msecs int) int {
	return int(C.wait_timed_semaphore((*C.sem_t)(unsafe.Pointer(s.semhandle)), C.int(msecs)))
}

func (s *Semaphore) TryWait() int {
	return int(C.try_wait_semaphore((*C.sem_t)(unsafe.Pointer(s.semhandle))))
}

func (s *Semaphore) Reset() int {
	return int(C.reset_semaphore((*C.sem_t)(unsafe.Pointer(s.semhandle))))
}

func (s *Semaphore) GetHandle() uintptr {
	return s.semhandle
}

func (s *Semaphore) Close() int {
	return 0
}
