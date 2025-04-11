package file

import "os"

type Status struct {
	File   *os.File
	Size   int64 //当前大小
	Expire int64 //过期时间
}

func NewStatus(f *os.File, expire int64) *Status {
	return &Status{File: f, Expire: expire}
}
