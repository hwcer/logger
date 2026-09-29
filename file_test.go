package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestFileWritePerformance 测试File日志器在不同并发量下的写入性能
func TestFileWritePerformance(t *testing.T) {
	// 禁用控制台输出以避免影响性能测试结果
	Console.Disable.Store(true)

	// 创建日志目录
	logDir := "./logs_perf"
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}

	// 测试不同的并发量
	concurrencyLevels := []int{100, 500, 1000, 2000, 5000}

	for _, goroutineCount := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", goroutineCount), func(t *testing.T) {
			// 创建File日志器实例，设置通道容量为1000
			f := NewFile(logDir, 1000)
			// 设置较大的文件大小限制，避免测试过程中触发文件切割
			f.SetFileSize(1000) // 1000MB

			// 用于统计写入的日志数量
			var counter int64
			var mutex sync.Mutex

			// 测试持续时间
			duration := 5 * time.Second

			// 记录开始时间
			startTime := time.Now()
			endTime := startTime.Add(duration)

			// 创建多个goroutine并发写入日志
			var writeWg sync.WaitGroup

			// 启动工作goroutine
			for g := range goroutineCount {
				writeWg.Add(1)
				go func(goroutineID int) {
					defer writeWg.Done()

					// 创建测试日志消息
					msg := &Message{
						Level:   LevelTrace,
						Time:    time.Now(),
						Content: "This is a performance test log message. This is a performance test log message.",
					}

					// 持续写入直到达到测试时间，每50毫秒写入一条，更接近实际业务场景
					for time.Now().Before(endTime) {
						f.Write(msg)
						mutex.Lock()
						counter++
						mutex.Unlock()

						// 每50毫秒写入一条日志
						time.Sleep(50 * time.Millisecond)
					}
				}(g)
			}

			// 等待所有写入goroutine完成
			writeWg.Wait()

			// 关闭日志器，确保所有缓冲区数据都被写入
			f.Close()

			// 计算实际测试持续时间
			actualDuration := time.Since(startTime)

			// 计算每秒写入的日志数量
			writesPerSecond := float64(counter) / actualDuration.Seconds()

			// 输出性能测试结果
			t.Logf("Performance Test Results for %d goroutines:", goroutineCount)
			t.Logf("- Total logs written: %d", counter)
			t.Logf("- Test duration: %.2f seconds", actualDuration.Seconds())
			t.Logf("- Estimated writes per second: %.2f", writesPerSecond)
		})
	}
}

// 🔴 回归:文件惰性创建 + SetFileName 换名。
// 曾因 NewFile 启动即按默认名建 log.log,换名要等默认 expire(下月1号),整月日志
// 落在误名文件——现在 NewFile 不建文件,常规时序(先 SetFileName 再写日志)
// 直接按正确名建,log.log 从源头不存在。
func TestSetFileNameRotatesImmediately(t *testing.T) {
	dir := t.TempDir()
	f := NewFile(dir)
	defer f.Close()
	f.SetFileName(func() (name, backup string, expire int64) {
		return "myapp.log", "", 0
	})
	f.Write(&Message{Level: LevelFatal, Content: "hello"}) //Fatal 走同步落盘路径,断言无需等待

	if _, err := os.Stat(filepath.Join(dir, "log.log")); !os.IsNotExist(err) {
		t.Fatal("默认名 log.log 不应存在(惰性创建,写入时 formatter 已就位)")
	}
	b, err := os.ReadFile(filepath.Join(dir, "myapp.log"))
	if err != nil || !strings.Contains(string(b), "hello") {
		t.Fatalf("日志必须直接写入新名文件:%v", err)
	}
}
