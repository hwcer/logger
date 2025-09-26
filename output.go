package logger

import (
	"fmt"
)

// Output Output输出时是否对字体染色
type Output interface {
	Write(message *Message)
	Close() error
}

func (log *Logger) SetOutput(name string, output Output) error {
	if _, ok := log.outputs[name]; ok {
		return fmt.Errorf("adapter name exist:%v", name)
	}
	log.mutex.Lock()
	defer log.mutex.Unlock()
	dict := make(map[string]Output)
	for k, v := range log.outputs {
		dict[k] = v
	}
	dict[name] = output
	log.outputs = dict
	return nil
}

func (log *Logger) DelOutput(name string) {
	if _, ok := log.outputs[name]; !ok {
		return
	}
	log.mutex.Lock()
	defer log.mutex.Unlock()
	dict := make(map[string]Output)
	for k, v := range log.outputs {
		if k != name {
			dict[k] = v
		}
	}
	log.outputs = dict
}
