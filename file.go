package logger

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hwcer/logger/file"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func NewFile(path string) *File {
	f := &File{logsPath: path}
	f.backup = file.NewBackup()
	return f
}

type File struct {
	limit    int64                           //文件大小(byte),0：不需要按容量切分
	backup   *file.Backup                    //备份锁
	status   *file.Status                    //文件状态
	layout   string                          //日期标记
	fileName file.NameFormatter              //日志名规则
	logsPath string                          //日志目录
	Sprintf  func(*Message) *strings.Builder //格式化message
}

func (this *File) GetFileName() (name string, expire int64) {
	if this.fileName != nil {
		return this.fileName()
	}
	return file.NameFormatterDefault()
}

func (this *File) MayBackup() error {
	if this.mayNeedBackup() {
		return this.backup.Handle(this.createFile)
	}
	return nil
}

// mayNeedBackup 是否需要开始备份
func (this *File) mayNeedBackup() bool {
	if this.status == nil {
		return true
	}
	if this.limit > 0 && this.status.Size >= this.limit {
		return true
	}
	if this.status.Expire > 0 && this.status.Expire < time.Now().Unix() {
		return true
	}
	return false
}

// SetFileSize 设置文件大小(M)，默认无限制
func (this *File) SetFileSize(n int64) {
	this.limit = n * 1024 * 1024
}

// SetFileName 设置日志文件名,  前缀(string) 或者 fileNameFormatter
func (this *File) SetFileName(f file.NameFormatter) {
	this.fileName = f
}

func (this *File) Write(msg *Message) (err error) {
	var b *strings.Builder
	if this.Sprintf != nil {
		b = this.Sprintf(msg)
	} else {
		b = msg.Sprintf()
	}
	b.WriteString("\n")
	//_, err = this.file.Write([]byte(txt))
	return this.write(b)
}

var errFileStatusNil = errors.New("file status is nil")

func (this *File) write(b *strings.Builder) (err error) {
	if err = this.MayBackup(); err != nil {
		return
	}
	status := this.status
	if status == nil {
		return errFileStatusNil
	}
	var n int64
	sb := bytes.NewBuffer([]byte(b.String()))
	if n, err = sb.WriteTo(status.File); err == nil && n > 0 {
		atomic.AddInt64(&status.Size, n)
	}
	return
}

func (this *File) createFile() (err error) {
	// Open the log file
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	var path string
	if path, err = filepath.Abs(this.logsPath); err != nil {
		return
	}

	if err = this.pathExists(path); err != nil {
		return
	}

	if err = this.backupFile(); err != nil {
		return
	}

	var perm int64
	perm, err = strconv.ParseInt("0777", 8, 64)
	if err != nil {
		return
	}
	name, expire := this.GetFileName()

	fd, err := os.OpenFile(filepath.Join(path, name), os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(perm))
	if err != nil {
		return err
	}
	fd.Fd()
	_ = os.Chmod(path, os.FileMode(perm))
	if err = fd.Sync(); err != nil {
		return
	}
	this.status = file.NewStatus(fd, expire)
	return
}

func (this *File) backupFile() (err error) {
	if this.status == nil {
		return nil
	}
	var status *file.Status
	status, this.status = this.status, nil
	_ = status.File.Close()
	name := status.File.Name()
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(filepath.Base(name), ext)
	path := filepath.Dir(name)
	for i := 1; ; i++ {
		s := strconv.Itoa(10000 + i)
		s = strings.TrimPrefix(s, "1")
		filename := filepath.Join(path, fmt.Sprintf("%s.%s%s", base, s, ext))
		if !this.fileExists(filename) {
			return os.Rename(name, filename)
		}
	}
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
