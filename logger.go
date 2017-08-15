package nano

import (
	"log"
	"os"
)

//Logger represents  the log interface
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
