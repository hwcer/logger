package logger

import (
	"sync"
	"testing"
	"time"
)

var wg = sync.WaitGroup{}

func TestFile(t *testing.T) {
	Console.Disable = true
	f := NewFile("./logs")
	f.SetFileSize(1)
	_ = SetOutput("file", f)
	SetLevel(LevelDebug)
	wg.Add(1)
	for i := 0; i < 100; i++ {
		go testAlert()
	}
	time.AfterFunc(time.Second, func() {
		wg.Done()
	})
	wg.Wait()
}

func testAlert() {
	wg.Add(1)
	defer wg.Done()
	t := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-t.C:
			Alert("testtesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttesttest")
		}
	}
}
