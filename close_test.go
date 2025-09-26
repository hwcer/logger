package logger

import (
	"sync"
	"testing"
	"time"
)

// TestCloseConcurrent 测试Close方法在并发环境下的安全性
func TestCloseConcurrent(t *testing.T) {
	log := New()
	// 添加一个控制台输出，确保有输出需要关闭
	log.SetOutput("console", Console)

	// 并发调用Close多次
	var wg sync.WaitGroup
	const goroutineCount = 100

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := log.Close()
			if err != nil {
				t.Errorf("Close returned error: %v", err)
			}
		}()
	}

	wg.Wait()
	
	// 验证再次调用Close不会导致panic或错误
	err := log.Close()
	if err != nil {
		t.Errorf("Subsequent Close returned error: %v", err)
	}
}

// TestCloseIdempotent 测试Close方法的幂等性
func TestCloseIdempotent(t *testing.T) {
	log := New()
	log.SetOutput("console", Console)

	// 第一次调用
	err1 := log.Close()
	if err1 != nil {
		t.Errorf("First Close returned error: %v", err1)
	}

	// 第二次调用应该返回nil
	err2 := log.Close()
	if err2 != nil {
		t.Errorf("Second Close returned error: %v", err2)
	}

	// 第三次调用也应该返回nil
	err3 := log.Close()
	if err3 != nil {
		t.Errorf("Third Close returned error: %v", err3)
	}
}

// 模拟一个会被多次调用Close的真实场景
func TestRealWorldCloseScenario(t *testing.T) {
	log := New()
	log.SetOutput("console", Console)

	// 模拟应用程序中可能多次尝试关闭日志的情况
	done := make(chan struct{})
	
	// 一个goroutine在某个时刻调用Close
	go func() {
		time.Sleep(10 * time.Millisecond)
		err := log.Close()
		if err != nil {
			t.Errorf("Background Close returned error: %v", err)
		}
		done <- struct{}{}
	}()

	// 主线程也尝试调用Close
	err := log.Close()
	if err != nil {
		t.Errorf("Main thread Close returned error: %v", err)
	}

	// 等待后台goroutine完成
	<-done
}