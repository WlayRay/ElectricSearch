package util

import (
	"log"
	"os"
)

var Log = log.New(os.Stdout, "[github.com/WlayRay/ElectricSearch]", log.Llongfile|log.Ldate|log.Ltime)
