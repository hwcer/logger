package logger

import (
	"fmt"
	"runtime"
	"strings"
)

var Console = &console{colorful: true}

func init() {
	//简化默认控制台输出
	Console.Sprintf = func(message *Message) *strings.Builder {
		b := strings.Builder{}
		b.WriteString(message.Content)
		return &b
	}
	if runtime.GOOS == "windows" {
		Console.colorful = false
	}
}

type console struct {
	Disable  bool
	Sprintf  func(*Message) *strings.Builder
	colorful bool
}

func (c *console) Name() string {
	return "_logger_console_name"
}

func (c *console) Write(msg *Message) {
	if c.Disable {
		return
	}
	var txt string
	level := msg.Level
	var b *strings.Builder
	if c.Sprintf != nil {
		b = c.Sprintf(msg)
	} else {
		b = msg.Sprintf()
	}
	txt = b.String()

	if c.colorful {
		txt = level.Brush(txt)
	}
	if msg.Stack != "" {
		txt = strings.Join([]string{txt, msg.Stack}, "\n")
	}
	_, _ = fmt.Println(txt)
}
