package logger

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// fileSystem 封装文件系统相关字段，方便创建和还原

type fileSystem struct {
	file           *os.File      // 文件句柄
	size           int64         // 当前大小
	expire         int64         // 过期时间（按日期切分）
	backup         string        // 备份名后缀   name.backup.index
	bufferedWriter *bufio.Writer // 缓冲写入器
}

type fileNameFormatter func() (name, backup string, expire int64)

// FileNameFormatterDefault 默认日志文件,每日一份
func FileNameFormatterDefault() (name, backup string, expire int64) {
	t := time.Now()
	r := time.Date(t.Year(), t.Month(), 0, 0, 0, 0, 0, t.Location())
	backup = t.Format("200601")
	name = "log.log"
	n := r.AddDate(0, 1, 0)
	expire = n.Unix() - 1
	return
}

func NewFile(path string, cap ...int) *File {
	f := &File{
		path:  path,
		limit: 0, // 默认不需要按容量切分
		index: 1,
	}
	if len(cap) > 0 {
		f.writer = make(chan *strings.Builder, cap[0])
	} else {
		f.writer = make(chan *strings.Builder, 1000)
	}

	f.fileNameFormatter = FileNameFormatterDefault
	f.wg.Add(1)
	go f.process()
	return f
}

type File struct {
	wg                sync.WaitGroup                  //等待组，用于优雅关闭
	fs                *fileSystem                     // 文件系统对象
	path              string                          //日志目录
	limit             int64                           //文件大小(byte),0：不需要按容量切分
	index             int                             //备份文件后缀
	Sprintf           func(*Message) *strings.Builder //格式化message
	writer            chan *strings.Builder           //写通道
	fileNameFormatter fileNameFormatter               //日志名规则
}

// SetFileSize 设置文件大小(M)，默认无限制
// 注意：该方法只应在初始化时调用
func (f *File) SetFileSize(n int64) {
	// limit字段仅在初始化时设置，无需并发保护
	f.limit = n * 1024 * 1024
}

// SetFileName 设置日志文件名,  前缀(string) 或者 fileNameFormatter
// 注意：该方法只应在初始化时调用
func (f *File) SetFileName(fileNameFormatterFunc fileNameFormatter) {
	// fileName字段仅在初始化时设置，无需并发保护
	f.fileNameFormatter = fileNameFormatterFunc
}

func (f *File) Write(msg *Message) {
	var b *strings.Builder

	if f.Sprintf != nil {
		b = f.Sprintf(msg)
	} else {
		b = msg.Sprintf()
	}
	b.WriteString("\n")

	// 阻塞模式写入，确保所有日志都能被处理
	f.writer <- b
}

// Close 优雅关闭日志文件
func (f *File) Close() error {
	// 关闭writer通道发送关闭信号
	// 注意：不再需要单独的close通道，writer通道的关闭信号已足够
	close(f.writer)

	// 等待process协程完成资源清理工作
	// 注意：资源清理的主要逻辑在process的defer函数中完成
	f.wg.Wait()

	// 不需要在这里再次清理资源，因为process协程的defer块已经完成了所有必要的资源清理
	return nil
}

func (f *File) process() {
	defer f.wg.Done()
	defer func() {
		if f.fs != nil && f.fs.bufferedWriter != nil {
			// 在关闭bufferedWriter前刷新缓冲区
			_ = f.fs.bufferedWriter.Flush()
			// 注意：关闭bufferedWriter时不需要单独关闭底层的os.File
			// 因为当设置bufferedWriter为nil后，垃圾回收会处理底层资源
			f.fs.bufferedWriter = nil
			f.fs.file = nil // 同时将file设置为nil以避免重复关闭
		}
	}()

	// 持续处理writer通道中的消息，直到通道关闭
	// 注意：writer通道的关闭信号已足够用于检测关闭事件
	for {
		b, ok := <-f.writer
		if !ok {
			// writer通道已关闭，处理完所有消息后退出
			// defer函数会自动执行完整的资源清理
			return
		}
		// 正常处理日志消息
		if f.mayNeedBackup() {
			_ = f.createFile()
		}
		f.writeFile(b)
	}
}

func (f *File) writeFile(b *strings.Builder) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("logger write file recover error:%v", e)
		}
	}()

	if f.fs == nil || f.fs.bufferedWriter == nil {
		return
	}

	// 直接写入缓冲写入器，避免不必要的转换
	if n, err := f.fs.bufferedWriter.WriteString(b.String()); err != nil && n > 0 {
		fmt.Printf("logger write file WriteString error:%v", err)
	} else if n > 0 {
		f.fs.size += int64(n)

		// 定期刷新缓冲区，但不要每次都刷新
		if f.fs.bufferedWriter.Available() < len(b.String())*2 {
			_ = f.fs.bufferedWriter.Flush()
		}
	}
}

// mayNeedBackup 是否需要开始备份
func (f *File) mayNeedBackup() bool {
	// 所有字段访问都在同一个goroutine中，无需锁保护
	if f.fs == nil {
		return true
	}
	if f.fs.file == nil {
		return true
	}
	if f.limit > 0 && f.fs.size >= f.limit {
		return true
	}
	if f.fs.expire > 0 && f.fs.expire < time.Now().Unix() {
		return true
	}
	return false
}

func (f *File) createFile() (err error) {
	// 所有操作都在同一个goroutine中，无需锁保护

	// 保存旧的文件系统对象，用于失败时恢复
	oldFS := f.fs
	defer func() {
		if err != nil {
			fmt.Printf("logger create file recover error:%v", err)
			// 如果没有旧文件系统，则触发panic
			if oldFS == nil {
				panic(fmt.Sprintf("critical error: cannot initialize log file system, %v", err))
			}
		}
	}()
	// 确保在尝试创建新文件前，先保存备份相关信息
	name, backup, expire := f.fileNameFormatter()
	path, err := filepath.Abs(f.path)
	if err != nil {
		return
	}
	if err = f.pathExists(path); err != nil {
		return
	}

	// 备份旧文件
	f.backupFile(oldFS)
	oldFS = nil //备份后文件系统已经被释放不可以重新使用

	var perm int64
	perm, err = strconv.ParseInt("0777", 8, 64)
	if err != nil {
		return
	}

	fd, err := os.OpenFile(filepath.Join(path, name), os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(perm))
	if err != nil {
		return
	}

	// 获取文件信息以设置正确的size
	fi, err := fd.Stat()
	if err != nil {
		_ = fd.Close()
		return
	}

	// 新文件创建成功，创建新的文件系统对象
	newFS := &fileSystem{
		file:           fd,
		size:           fi.Size(),
		expire:         expire,
		backup:         backup,
		bufferedWriter: bufio.NewWriterSize(fd, 4*1024*1024),
	}

	// 替换旧的文件系统对象
	f.fs = newFS
	return
}

// backupFile 使用静默方式，如果失败新的文件系统也只会继续使用当前文件
func (f *File) backupFile(fs *fileSystem) {
	if fs == nil {
		return
	}
	var err error
	defer func() {
		if err != nil {
			fmt.Printf("logger backup file error:%v", err)
		}
	}()
	// 先保存文件名
	name := fs.file.Name()
	// 先刷新缓冲区
	_ = fs.bufferedWriter.Flush()

	// 关闭文件以准备重命名
	if err = fs.file.Close(); err != nil {
		return
	}

	// 备份操作不需要修改f.fs，因为我们只在createFile中替换它
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(filepath.Base(name), ext)
	if fs.backup != "" {
		base = fmt.Sprintf("%s.%s", base, fs.backup)
	}
	path := filepath.Dir(name)
	for i := f.index + 1; ; i++ {
		s := strconv.Itoa(10000 + i)
		s = strings.TrimPrefix(s, "1")
		filename := filepath.Join(path, fmt.Sprintf("%s.%s%s", base, s, ext))
		if f.fileExists(filename) {
			continue
		}
		if err = os.Rename(name, filename); err == nil {
			f.index = i
			break
		}
	}

	return
}

func (f *File) fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func (f *File) pathExists(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("path not dir:%v", path)
	}
	return nil
}
