package logger

import (
	"sync"
	"testing"
)

// 🔴 回归:运行期 SetLevel/SetOutput 与 Write 热路径必须无数据竞争——
// revert 曾把这里改回裸读写,-race 实报 logger.go 竞争;运行期改配置在本生态是
// 真实用法(cosgo SIGHUP 关控制台)。atomic 发布后本测试在 -race 下必须干净
func TestConcurrentConfigAndWrite(t *testing.T) {
	log := New()
	name := "_test_output"
	if err := log.SetOutput(name, Console); err != nil {
		t.Fatalf("SetOutput: %v", err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	//写日志热路径
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					log.Info("concurrent config test")
				}
			}
		}()
	}
	//并发改配置
	for i := 0; i < 200; i++ {
		log.SetLevel(Level(int32(i%7) + 1))
		if i%2 == 0 {
			_ = log.SetOutput(name, Console)
		} else {
			log.RemoveOutput(name)
		}
	}
	//恢复一个输出,避免写协程空转后半程
	_ = log.SetOutput(name, Console)
	close(stop)
	wg.Wait()
	log.RemoveOutput(name)
}
