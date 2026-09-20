package logger

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 由Trae AI负责优化

// fileSystem 封装文件系统相关字段，方便创建和还原

type fileSystem struct {
	file           *os.File      // 文件句柄
	size           int64         // 当前大小
	expire         int64         // 过期时间（按日期切分）
	backup         string        // 备份名后缀,为空时不会自动备份(比如name中已经包含了备份名,日期)   name.backup.index
	bufferedWriter *bufio.Writer // 缓冲写入器
}

type fileNameFormatter func() (name, backup string, expire int64)

// FileNameFormatterDefault 默认日志文件,每日一份
func FileNameFormatterDefault() (name, backup string, expire int64) {
	t := time.Now()
	backup = t.Format("200601")
	name = "log.log"
	n := t.AddDate(0, 1, 0)
	r := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, t.Location())
	expire = r.Unix()
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
	f.bufferFlushInterval.Store(int64(time.Second)) //默认一秒刷新一次
	f.fileNameFormatter = FileNameFormatterDefault
	//启动即同步建文件:此前 f.fs 要等第一个刷新周期(默认1s)的定时器才创建,
	//期间 writeFile 判空直接丢弃——进程启动最前面的日志(往往含启动关键信息)无痕丢失。
	//此时 process 协程尚未启动,无并发;失败时 fs 保持 nil,后续每个定时周期自动重试
	f.createFile()
	f.wg.Add(1)
	go f.process()
	return f
}

type File struct {
	wg                  sync.WaitGroup                  //等待组，用于优雅关闭
	fs                  *fileSystem                     // 文件系统对象
	path                string                          //日志目录
	limit               int64                           //文件大小(byte),0：不需要按容量切分
	index               int                             //备份文件后缀
	Sprintf             func(*Message) *strings.Builder //格式化message
	writer              chan *strings.Builder           //写通道
	fileNameFormatter   fileNameFormatter               //日志名规则
	bufferFlushInterval atomic.Int64                    //缓冲区时间间隔(允许运行时并发修改)
	mu                  sync.Mutex                      //串行化 fs/缓冲写访问:process 协程与 Fatal/Panic 同步落盘路径共享
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

// SetFlushInterval 设置缓冲区刷新间隔
// 注意：该方法可以在运行时调用，会在下一次定时器触发时生效
func (f *File) SetFlushInterval(interval time.Duration) {
	if interval <= 0 {
		return // 不允许设置非正的刷新间隔
	}
	f.bufferFlushInterval.Store(int64(interval))
}

func (f *File) Write(msg *Message) {
	var b *strings.Builder

	if f.Sprintf != nil {
		b = f.Sprintf(msg)
	} else {
		b = msg.Sprintf()
	}
	b.WriteString("\n")

	//Fatal/Panic 同步落盘:调用方随后通常 os.Exit(进程终止),走缓冲通道大概率
	//来不及被 process 协程消费,4MB bufio 也未 flush——最不能丢的日志反而必丢。
	//同步写+立即 Flush,不再入队(避免 process 侧重复落盘)
	if msg.Level >= LevelPanic {
		f.writeSync(b)
		return
	}

	// 阻塞模式写入，确保所有日志都能被处理
	f.writer <- b
}

// writeSync Fatal/Panic 专用:绕过缓冲通道直接写文件并立即 Flush。
//
// ⚠️ mu 的竞争面说明(为什么正常路径不心疼它):writeFile 仅由 process 单协程调用,
// 正常流量下 mu 无竞争(uncontended mutex ~20ns);唯一对手是本函数,而它只在
// Fatal/Panic 时出现。锁保护的是"process vs 直写路径"共享的 bufio——通道只管
// 投递,管不到绕过通道的旁路。持锁 Flush 的磁盘 syscall 会短暂停顿管道,
// 换来"Panic 日志严格排在既有异步日志之后"的顺序语义,值得。
func (f *File) writeSync(b *strings.Builder) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fs == nil || f.fs.bufferedWriter == nil {
		//🔴 启动建文件失败期间(目录不可写/EMFILE)来的 Fatal/Panic 也不能无痕消失:
		//降级 stderr,与 createFile 失败时的策略一致
		fmt.Fprint(os.Stderr, b.String())
		return
	}
	n, err := f.fs.bufferedWriter.WriteString(b.String())
	if err != nil {
		fmt.Printf("logger write file sync error:%v", err)
		return
	}
	f.fs.size += int64(n)
	//Fatal 是最不能丢的日志,Flush 失败必须可见:盘满/句柄失效时静默=日志无痕丢失
	if err = f.fs.bufferedWriter.Flush(); err != nil {
		fmt.Printf("logger flush on fatal error:%v", err)
	}
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
		// 确保在退出前刷新缓冲区并释放资源
		f.mu.Lock()
		if f.fs != nil {
			if f.fs.bufferedWriter != nil {
				_ = f.fs.bufferedWriter.Flush()
			}
			if f.fs.file != nil {
				_ = f.fs.file.Close() //旧实现只 flush 不 Close,文件句柄泄漏(Windows 下还会锁住文件)
			}
			f.fs.bufferedWriter = nil
			f.fs.file = nil
		}
		f.mu.Unlock()
	}()

	// 创建定时器并确保在函数退出时停止
	timer := time.NewTimer(time.Duration(f.bufferFlushInterval.Load()))
	defer timer.Stop()

	// 持续处理writer通道中的消息，直到通道关闭
	for {
		select {
		case b, ok := <-f.writer:
			if !ok {
				return
			}
			f.writeFile(b)
		case <-timer.C:
			f.mu.Lock()
			if f.mayNeedBackup() {
				f.createFile()
			} else if f.fs != nil && f.fs.bufferedWriter != nil {
				_ = f.fs.bufferedWriter.Flush()
			}
			f.mu.Unlock()
			timer.Reset(time.Duration(f.bufferFlushInterval.Load()))
		}
	}
}

func (f *File) writeFile(b *strings.Builder) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("logger write file recover error:%v", e)
		}
	}()

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.fs == nil || f.fs.bufferedWriter == nil {
		return
	}

	// 直接写入缓冲写入器，避免不必要的转换
	// 出错一律可见并跳过记账(部分写入的字节宁可少记——mayNeedBackup 只会因此
	// 提前切分,不会漏切)
	if n, err := f.fs.bufferedWriter.WriteString(b.String()); err != nil {
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

func (f *File) createFile() {
	// 所有操作都在同一个goroutine中，无需锁保护
	var err error

	// 保存旧的文件系统对象，用于备份
	oldFS := f.fs
	defer func() {
		if err != nil {
			//创建失败降级为 stderr 告警 + 每个定时周期重试,不 panic——
			//目录不存在/EMFILE/权限变更等错误不该把整个进程打崩(旧实现 oldFS==nil 时 panic)
			fmt.Printf("logger create file error:%v\n", err)
		}
	}()
	// 确保在尝试创建新文件前，先保存备份相关信息
	name, backup, expire := f.fileNameFormatter()
	path, err := filepath.Abs(f.path)
	if err != nil {
		return
	}
	//目录不存在时自动创建(常见运维失误:没建日志目录);存在但不是目录仍失败
	if err = f.ensurePath(path); err != nil {
		return
	}

	// 备份旧文件
	f.backupFile(oldFS)
	oldFS = nil //备份后文件系统已经被释放不可以重新使用

	fd, err := os.OpenFile(filepath.Join(path, name), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		return
	}

	// 获取文件信息以设置正确的size
	fi, err := fd.Stat()
	if err != nil {
		_ = fd.Close()
		return
	}
	err = nil

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
}

// backupFile 使用静默方式，如果失败新的文件系统也只会继续使用当前文件
func (f *File) backupFile(fs *fileSystem) {
	if fs == nil || fs.backup == "" {
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
	base = fmt.Sprintf("%s.%s", base, fs.backup)

	path := filepath.Dir(name)
	//最多尝试100个备份名,全部失败时放弃备份,原文件保持原名继续追加
	for i := range 100 {
		n := f.index + i + 1
		s := strconv.Itoa(10000 + n)
		s = strings.TrimPrefix(s, "1")
		filename := filepath.Join(path, fmt.Sprintf("%s.%s%s", base, s, ext))
		if f.fileExists(filename) {
			continue
		}
		if err = os.Rename(name, filename); err == nil {
			f.index = n
			break
		}
	}
}

func (f *File) fileExists(file string) bool {
	_, err := os.Stat(file)
	return !errors.Is(err, os.ErrNotExist)
}

// ensurePath 日志目录存在(或是自动创建)时返回 nil;路径被文件占用时报错
func (f *File) ensurePath(path string) error {
	stat, err := os.Stat(path)
	if err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("path not dir:%v", path)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.MkdirAll(path, 0755) //0777 在 umask=0 的容器里是全局可写目录
}
