package file

import "sync/atomic"

type Backup struct {
	n int32
	c chan struct{}
}

func NewBackup() *Backup {
	return &Backup{c: make(chan struct{})}
}

func (b *Backup) Handle(f func() error) (err error) {
	n, c := b.n, b.c
	if n > 1 {
		return
	}
	if !atomic.CompareAndSwapInt32(&b.n, 0, 1) {
		<-c
		return
	}
	defer func() {
		b.n = 2
		b.c = make(chan struct{})
		close(c)
		b.n = 0
	}()
	return f()
}
