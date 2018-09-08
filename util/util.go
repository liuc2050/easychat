package util

import (
	"io"
	"log"
	"sync"
)

type NoCopy struct{}

func (n *NoCopy) Lock() {}

//Stopper 用于控制goroutine优雅结束
type Stopper struct {
	noCopy NoCopy
	StopCh chan struct{}
	N      *sync.WaitGroup
}

func NewStopper() *Stopper {
	return &Stopper{StopCh: make(chan struct{}), N: &sync.WaitGroup{}}
}

func (s *Stopper) Stop() {
	if s == nil {
		return
	}
	close(s.StopCh)
	s.N.Wait()
}

type StringReader string

func (r *StringReader) Read(b []byte) (n int, err error) {
	n = copy(b, *r)
	*r = (*r)[n:]
	if n == 0 {
		err = io.EOF
	}
	return
}

func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, val := range a {
		if val != b[i] {
			return false
		}
	}
	return true
}

func TraceFunc(l *log.Logger, name string) func() {
	l.Printf("%s enter.", name)
	return func() {
		l.Printf("%s exit.", name)
	}
}
