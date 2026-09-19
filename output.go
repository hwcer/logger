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
	if _, ok := log.outputs[name]; ok {
		return fmt.Errorf("adapter name exist:%v", name)
	}
	dict := maps.Clone(log.outputs)
	dict[name] = output
	log.outputs = dict
	return nil
}

func (log *Logger) RemoveOutput(name string) {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	if _, ok := log.outputs[name]; !ok {
		return
	}
	dict := maps.Clone(log.outputs)
	delete(dict, name)
	log.outputs = dict
}
