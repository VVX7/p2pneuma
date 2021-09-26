package util

import "log"

func DebugLogf(format string, v ...interface{}) {
	if *DebugMode {
		log.Printf(format, v...)
	}
}

func DebugLog(v ...interface{}) {
	if *DebugMode {
		log.Print(v...)
	}
}
