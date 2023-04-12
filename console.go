package logger

import (
	"os"
	"runtime"
	"strings"
	"sync"
)

func NewConsole() *Console {
	return &Console{colorful: true}
}

type Console struct {
	sync.Mutex
	Sprintf  func(*Message) string
	colorful bool
}

func (c *Console) Init() (err error) {
	if runtime.GOOS == "windows" {
		c.colorful = false
	}
	return
}

func (c *Console) Write(msg *Message) error {
	var txt string
	level := msg.Level
	if c.Sprintf != nil {
		txt = c.Sprintf(msg)
	} else {
		txt = msg.String()
	}
	if c.colorful {
		txt = level.Brush(txt)
	}
	if level >= LevelError {
		txt = strings.Join([]string{txt, msg.Stack}, "\n")
	}
	return c.printlnConsole(txt)
}

func (c *Console) printlnConsole(msg string) (err error) {
	c.Lock()
	defer c.Unlock()
	_, err = os.Stdout.Write(append([]byte(msg), '\n'))
	return
}
