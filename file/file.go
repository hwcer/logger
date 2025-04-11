package file

import "time"

type NameFormatter func() (name string, expire int64)

// NameFormatterDefault 默认日志文件,每日一份
func NameFormatterDefault() (name string, expire int64) {
	t := time.Now()
	r := time.Date(t.Year(), t.Month(), 0, 0, 0, 0, 0, t.Location())
	name = t.Format("200601") + ".log"
	n := r.AddDate(0, 1, 0)
	expire = n.Unix() - 1
	return
}
