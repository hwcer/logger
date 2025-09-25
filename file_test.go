package logger

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// TestFileWritePerformance 测试File日志器在不同并发量下的写入性能
func TestFileWritePerformance(t *testing.T) {
	// 禁用控制台输出以避免影响性能测试结果
	Console.Disable = true

	// 创建日志目录
	logDir := "./logs_perf"
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}

	// 测试不同的并发量
	concurrencyLevels := []int{1, 2, 5, 10, 20, 50}
	
	for _, goroutineCount := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", goroutineCount), func(t *testing.T) {
			// 创建File日志器实例，设置较大的通道容量以避免写入阻塞
			f := NewFile(logDir, 100000)
			// 设置较大的文件大小限制，避免测试过程中触发文件切割
			f.SetFileSize(1000) // 1000MB

			// 用于统计写入的日志数量
			var counter int64
			var mutex sync.Mutex
			
			// 用于记录已写入但可能尚未处理的日志数量(估算队列长度)
				var sentCount int64
				var processedCount int64
				var maxQueueLength int
				var counterMutex sync.Mutex
				
				// 设置全局回调函数，用于更新已处理计数
				originalCallback := onLogProcessed
				onLogProcessed = func() {
					counterMutex.Lock()
					processedCount++
					counterMutex.Unlock()
				}
				defer func() {
					// 测试结束后重置回调函数
					onLogProcessed = originalCallback
				}()
				
				// 启动队列长度估算goroutine
				go func() {
					for {
						counterMutex.Lock()
						// 估算队列长度 = 已发送 - 已处理
						currentQueueLength := int(sentCount - processedCount)
						if currentQueueLength > maxQueueLength {
							maxQueueLength = currentQueueLength
						}
						counterMutex.Unlock()
						time.Sleep(5 * time.Millisecond)
					}
				}()

			// 测试持续时间
			duration := 5 * time.Second

			// 记录开始时间
			startTime := time.Now()
			endTime := startTime.Add(duration)

			// 创建多个goroutine并发写入日志
			var writeWg sync.WaitGroup

			// 启动工作goroutine
			for g := 0; g < goroutineCount; g++ {
				writeWg.Add(1)
				go func(goroutineID int) {
					defer writeWg.Done()

					// 创建测试日志消息
					msg := &Message{
						Level:   LevelTrace,
						Time:    time.Now(),
						Content: "This is a performance test log message. This is a performance test log message.",
					}

					// 持续写入直到达到测试时间
						for time.Now().Before(endTime) {
							f.Write(msg)
							mutex.Lock()
							counter++
							mutex.Unlock()
							// 更新已发送计数
							counterMutex.Lock()
							sentCount++
							counterMutex.Unlock()
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
			t.Logf("- Maximum queue length: %d", maxQueueLength)
		})
	}
}
