package nano

import (
	"log"
	"os"
)

type Logger interface {
	Println(v ...interface{})
	Fatal(v ...interface{})
}

// Default logger
var logger Logger = log.New(os.Stderr, "", log.LstdFlags|log.Llongfile)

// SetLogger rewrites the default logger
func SetLogger(l Logger) {
	if l != nil {
		logger = l
	}
}

// debugPrintln enable output when running under debug mode
func debugPrintln(v ...interface{}) {
	if env.debug {
		logger.Println(v...)
	}
}
