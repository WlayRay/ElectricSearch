package util

import (
	"log"
	"os"
)

var Log = log.New(os.Stdout, "[ElectricSearch]", log.Lshortfile|log.Ldate|log.Ltime)
