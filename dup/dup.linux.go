//go:build linux

package dup

func Dup(from int, to int) (err error) {
	return syscall.Dup2(oldfd, newfd)
}
