package main

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Logger struct {
	*zerolog.Logger
	cLogFile *os.File
}

var GLogger *Logger = nil

//we don't really need a file to log on containers since we capture the output
func NewLogger(enableFileLog, isGlobal bool) *Logger {
	var writers []io.Writer
	var file *os.File
	if enableFileLog {
		file, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			log.Panic("Couldn't open log")
		}

		writers = append(writers, file)
	}

	writers = append(writers, zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	mw := io.MultiWriter(writers...)

	logger := zerolog.New(mw).With().Timestamp().Logger()

	nlogger := &Logger{
		Logger:   &logger,
		cLogFile: file,
	}
	if isGlobal {
		GLogger = nlogger
	}
	return nlogger
}

func (m *Logger) CloseLogger() {
	if m.cLogFile != nil {
		m.cLogFile.Close()
	}
}
