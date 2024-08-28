package util

import (
	"log"
	"os"
)

var Log = log.New(os.Stdout, "[MiniES]", log.Lshortfile|log.Ldate|log.Ltime)
