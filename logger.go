package main

import (
	"github.com/go-logr/logr"
)

// Testing
type LoggerWrapper struct {
	log logr.Logger
}

func (l *LoggerWrapper) Println(v ...interface{}) {
	l.log.Info("internal error", v...)
}
