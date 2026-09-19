package logger

import (
	"fmt"
	"maps"
)

// Output Output输出时是否对字体染色
type Output interface {
	Write(message *Message)
	Close() error
}

func (log *Logger) SetOutput(name string, output Output) error {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	cur := *log.outputs.Load()
	if _, ok := cur[name]; ok {
		return fmt.Errorf("adapter name exist:%v", name)
	}
	dict := maps.Clone(cur)
	dict[name] = output
	log.outputs.Store(&dict)
	return nil
}

func (log *Logger) RemoveOutput(name string) {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	cur := *log.outputs.Load()
	if _, ok := cur[name]; !ok {
		return
	}
	dict := maps.Clone(cur)
	delete(dict, name)
	log.outputs.Store(&dict)
}
