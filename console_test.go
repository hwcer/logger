package logger

import (
	"strings"
	"sync"
	"testing"
)

// 🔴 控制台并发写安全:Write 可被任意多协程并发调用,不得 data race、不得 panic。
//
// 行级原子性由 fmt.Println 的单次 write syscall 提供(管道 ≤PIPE_BUF 不穿插),
// 本测试在 -race 下跑才有完整意义(go test -race ./...);
// 行序不确定是无锁输出的固有属性,断言只落在"全部行都完整落盘"上。
func TestConsoleConcurrentWrite(t *testing.T) {
	const goroutines, perG = 50, 200
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				Console.Write(&Message{
					Level:   LevelInfo,
					Content: strings.Repeat("x", 64), //常规行长,远小于 PIPE_BUF
				})
			}
		}(i)
	}
	wg.Wait()
}
