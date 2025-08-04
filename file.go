package logger

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type fileNameFormatter func() (name, backup string, expire int64)

// FileNameFormatter 默认日志文件,每日一份
func FileNameFormatter() (name, backup string, expire int64) {
	t := time.Now()
	r := time.Date(t.Year(), t.Month(), 0, 0, 0, 0, 0, t.Location())
	backup = t.Format("200601")
	name = "log.log"
	n := r.AddDate(0, 1, 0)
	expire = n.Unix() - 1
	return
}

func NewFile(path string, cap ...int) *File {
	f := &File{path: path}
	if len(cap) > 0 {
		f.writer = make(chan *strings.Builder, cap[0])
	} else {
		f.writer = make(chan *strings.Builder, 1000)
	}
	go f.process()
	return f
}

type File struct {
	path     string //日志目录
	file     *os.File
	size     int64                           //当前大小
	limit    int64                           //文件大小(byte),0：不需要按容量切分
	expire   int64                           //过期时间
	backup   string                          //备份名后缀   name.backup.index
	fileName fileNameFormatter               //日志名规则
	Sprintf  func(*Message) *strings.Builder //格式化message
	writer   chan *strings.Builder           //写通道
	index    int                             //备份文件后缀
}

func (this *File) GetFileName() (name, backup string, expire int64) {
	if this.fileName != nil {
		return this.fileName()
	}
	return FileNameFormatter()
}

//func (this *File) MayBackup() error {
//	if this.mayNeedBackup() {
//		return this.backup.Handle(this.createFile)
//	}
//	return nil
//}

// SetFileSize 设置文件大小(M)，默认无限制
func (this *File) SetFileSize(n int64) {
	this.limit = n * 1024 * 1024
}

// SetFileName 设置日志文件名,  前缀(string) 或者 fileNameFormatter
func (this *File) SetFileName(f fileNameFormatter) {
	this.fileName = f
}

func (this *File) Write(msg *Message) {
	var b *strings.Builder
	if this.Sprintf != nil {
		b = this.Sprintf(msg)
	} else {
		b = msg.Sprintf()
	}
	b.WriteString("\n")
	this.writer <- b
}

func (this *File) process() {
	defer func() {
		if this.file != nil {
			_ = this.file.Close()
		}
	}()
	for b := range this.writer {
		if this.mayNeedBackup() {
			this.createFile()
		}
		this.writeFile(b)
	}
}

func (this *File) writeFile(b *strings.Builder) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("logger write file recover error:%v", e)
		}
	}()
	sb := bytes.NewBuffer([]byte(b.String()))
	if n, err := sb.WriteTo(this.file); err != nil && n > 0 {
		fmt.Printf("logger write file WriteTo data:%s", sb.String())
		fmt.Printf("logger write file WriteTo error:%v", err)
	} else if n > 0 {
		this.size += n
	}
	return
}

// mayNeedBackup 是否需要开始备份
func (this *File) mayNeedBackup() bool {
	if this.file == nil {
		return true
	}
	if this.limit > 0 && this.size >= this.limit {
		return true
	}
	if this.expire > 0 && this.expire < time.Now().Unix() {
		return true
	}
	return false
}

func (this *File) createFile() {
	// Open the log file
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("logger create file recover error:%v", e)
		}
	}()
	path, err := filepath.Abs(this.path)
	if err != nil {
		fmt.Printf("logger create file filepath.Abs error:%v", err)
		return
	}

	if err = this.pathExists(path); err != nil {
		fmt.Printf("logger create file pathExists error:%v", err)
		return
	}

	if err = this.backupFile(); err != nil {
		fmt.Printf("logger create file pathExists error:%v", err)
		return
	}

	var perm int64
	perm, err = strconv.ParseInt("0777", 8, 64)
	if err != nil {
		fmt.Printf("logger create file perm error:%v", err)
		return
	}
	name, backup, expire := this.GetFileName()

	fd, err := os.OpenFile(filepath.Join(path, name), os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(perm))
	if err != nil {
		fmt.Printf("logger create file OpenFile error:%v", err)
		return
	}
	fd.Fd()
	_ = os.Chmod(path, os.FileMode(perm))
	if err = fd.Sync(); err != nil {
		fmt.Printf("logger create file fd.Sync error:%v", err)
		return
	}
	this.size = 0
	this.file = fd
	this.backup = backup
	this.expire = expire
	return
}

func (this *File) backupFile() (err error) {
	if this.file == nil {
		return nil
	}
	if err = this.file.Close(); err != nil {
		return
	}

	name := this.file.Name()
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(filepath.Base(name), ext)
	if this.backup != "" {
		base = fmt.Sprintf("%s.%s", base, this.backup)
	}
	path := filepath.Dir(name)
	for i := this.index + 1; ; i++ {
		s := strconv.Itoa(10000 + i)
		s = strings.TrimPrefix(s, "1")
		filename := filepath.Join(path, fmt.Sprintf("%s.%s%s", base, s, ext))
		if this.fileExists(filename) {
			continue
		}
		if err = os.Rename(name, filename); err == nil {
			this.index = i
			break
		}
	}
	return
}

func (this *File) fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func (this *File) pathExists(path string) error {
	f, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !f.IsDir() {
		return fmt.Errorf("path not dir:%v", path)
	}
	return nil
}
