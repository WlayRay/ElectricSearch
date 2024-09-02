package util

import (
	"log"
	"os"
)

var Log = log.New(os.Stdout, "[github.com/WlayRay/ElectricSearch/v1.0.0]", log.Llongfile|log.Ldate|log.Ltime)
