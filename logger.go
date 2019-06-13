package redis

type Logger interface {
	Print(v ...interface{})
}
