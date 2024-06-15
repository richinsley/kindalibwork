package main

import (
	sem "github.com/richinsley/kindalib/pkg/semaphore"
)

func main() {
	s := sem.NewSemaphore()
	s.Post()
	s.Wait()
	s.Close()
}
