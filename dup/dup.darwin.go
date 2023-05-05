//go:build darwin

package dup

func Dup(from int, to int) (err error) {
	return syscall.Dup2(from, to)
}
