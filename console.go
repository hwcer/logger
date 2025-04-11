package logger

import (
	"fmt"
	"runtime"
	"strings"
)

var Console = &console{colorful: true}

func init() {
	//简化默认控制台输出
	Console.Sprintf = func(message *Message) string {
		return message.Content
	}
	if runtime.GOOS == "windows" {
		Console.colorful = false
	}
}

type console struct {
	Disable  bool
	Sprintf  func(*Message) string
	colorful bool
}

func (c *console) Name() string {
	return "_logger_console_name"
}
func (c *console) Write(msg *Message) error {
	if c.Disable {
		return nil
	}
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
	_, err := fmt.Println(txt)
	return err
}
