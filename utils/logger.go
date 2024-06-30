package utils

import (
	"os"

	"github.com/rs/zerolog"
)

type LogLevel int

const (
	LogLevel_Trace   LogLevel = 0
	LogLevel_Debug   LogLevel = 1
	LogLevel_Info    LogLevel = 2
	LogLevel_Warning LogLevel = 3
	LogLevel_Error   LogLevel = 4
)

var log zerolog.Logger

func init() {
	file, err := os.OpenFile(
		"SplitFlapApp.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0664,
	)

	if err != nil {
		panic(err)
	}

	log = zerolog.New(file).With().Timestamp().Logger()
}

func Log(logLevel LogLevel, msg string, err error) {
	var logType *zerolog.Event
	switch logLevel {
	case LogLevel_Trace:
		logType = log.Trace()
	case LogLevel_Debug:
		logType = log.Debug()
	case LogLevel_Info:
		logType = log.Info()
	case LogLevel_Warning:
		logType = log.Warn()
	case LogLevel_Error:
		logType = log.Error()
		if err != nil {
			logType.Err(err)
			logType.Msg(err.Error())
		}
	default:
		panic("Log level not supported")
	}

	logType.Msg(msg)
}
