package log

import (
	"log"
	"os"
)

// Logger represents  the log interface
type Logger interface {
	Println(v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
}

func init() {
	SetLogger(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile))
}

var (
	Println func(v ...interface{})
	Fatal   func(v ...interface{})
	Fatalf  func(format string, v ...interface{})
)

// SetLogger rewrites the default logger
func SetLogger(logger Logger) {
	if logger == nil {
		return
	}
	Println = logger.Println
	Fatal = logger.Fatal
	Fatalf = logger.Fatalf
}
