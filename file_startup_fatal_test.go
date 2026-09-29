package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 🔴 P0 回归:首条日志必须即时建文件——此前要等第一个刷新周期(默认1s)才创建,
// 期间 writeFile 判空丢弃,进程启动最前面的日志(往往含启动关键信息)无痕丢失。
// 注意时序契约:文件**惰性创建**,NewFile 本身不建文件(SetFileName 还没来,建了
// 就是误名的 log.log);第一条 Write 到达时必须立即按当前 formatter 名建出
func TestNewFileCreatesFileImmediately(t *testing.T) {
	dir := t.TempDir()
	f := NewFile(dir)
	defer func() { _ = f.Close() }()

	name, _, _ := f.fileNameFormatter()
	if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
		t.Fatal("NewFile 不应预建文件(惰性创建,防误名 log.log)")
	}
	f.Write(&Message{Level: LevelFatal, Content: "first"}) //Fatal 同步落盘,即时建文件
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatalf("首条日志写入后文件应立即存在: %v", err)
	}
}

// 目录不存在时自动创建(常见运维失误:没建日志目录),且不再 panic 打崩进程
func TestNewFileCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "deep", "nested", "logs")
	f := NewFile(dir)
	defer func() { _ = f.Close() }()

	name, _, _ := f.fileNameFormatter()
	f.Write(&Message{Level: LevelFatal, Content: "first"})
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatalf("首条日志写入时缺失目录应被自动创建并建文件: %v", err)
	}
}

// 🔴 P0 回归:Fatal/Panic 同步落盘——调用方随后 os.Exit,缓冲通道来不及消费,
// 最不能丢的日志反而必丢。写完即应能在文件里读到,不等刷新周期
func TestFatalWriteSyncedToDisk(t *testing.T) {
	dir := t.TempDir()
	f := NewFile(dir)
	defer func() { _ = f.Close() }()

	f.Write(&Message{Level: LevelFatal, Content: "fatal-line", Time: time.Now()})
	f.Write(&Message{Level: LevelPanic, Content: "panic-line", Time: time.Now()})

	name, _, _ := f.fileNameFormatter()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("读日志文件: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "fatal-line") || !strings.Contains(s, "panic-line") {
		t.Fatalf("Fatal/Panic 应同步落盘(无需等待刷新周期),实际内容:\n%s", s)
	}
}
