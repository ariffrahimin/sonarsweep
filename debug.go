package main

import (
	"fmt"
	"os"
	"time"
)

var debugFile *os.File

func initDebug() {
	var err error
	debugFile, err = os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open debug log:", err)
	}
}

func debug(format string, args ...interface{}) {
	if debugFile == nil {
		initDebug()
	}
	if debugFile == nil {
		return
	}
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(debugFile, "%s %s\n", timestamp, msg)
	debugFile.Sync()
}

func closeDebug() {
	if debugFile != nil {
		debugFile.Close()
	}
}