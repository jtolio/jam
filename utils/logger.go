package utils

import (
	"log"
	"os"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

var DefaultLogger Logger = log.New(os.Stderr, "", log.LstdFlags)
