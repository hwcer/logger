package logger

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
)

var Console = &console{colorful: true}

func init() {
	if runtime.GOOS == "windows" {
		Console.colorful = false
	}
}

type console struct {
	// Disable 运行期可安全切换(cosgo SIGHUP 关控制台):atomic 保证与 Write 的
	// 并发读无 data race。行级原子性由 fmt.Println 的单次 write syscall 提供,
	// 行序不确定是无锁输出的固有属性
	Disable  atomic.Bool
	Sprintf  func(*Message) *strings.Builder
	colorful bool
}

func (c *console) Name() string {
	return "_logger_console_name"
}
func (c *console) Close() error {
	return nil
}
func (c *console) Write(msg *Message) {
	if c.Disable.Load() {
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
		txt = txt + "\n" + msg.Stack
	}
	_, _ = fmt.Println(txt)
}
