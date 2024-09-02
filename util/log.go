package util

import (
	"log"
	"os"
)

var Log = log.New(os.Stdout, "[ElectricSearch]", log.Llongfile|log.Ldate|log.Ltime)
