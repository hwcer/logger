package logger

import (
	"strings"
	"time"
)

const defaultTimeLayout = "2006-01-02 15:04:05-0700" // 日志输出默认格式

type Message struct {
	Time    time.Time
	Level   Level
	Stack   string
	Content string
}

func (this *Message) Sprintf() *strings.Builder {
	b := strings.Builder{}
	b.WriteString(this.Time.Format(defaultTimeLayout))
	b.WriteString(" [")
	b.WriteString(this.Level.String())
	b.WriteString("] ")
	b.WriteString(this.Content)
	return &b
}
