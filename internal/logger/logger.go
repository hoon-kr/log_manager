// Copyright 2024 JongHoon Shim and The log_manager Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux

/*
Package logger process log.
*/
package logger

import (
	"fmt"
	"strings"

	"github.com/hoon-kr/log_manager/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger interface
type Logger interface {
	InitializeLogger()
	FinalizeLogger()
	LogInfo(format string, args ...interface{})
	LogWarn(format string, args ...interface{})
	LogError(format string, args ...interface{})
	LogDebug(format string, args ...interface{})
	LogPanic(format string, args ...interface{})
	LogFatal(format string, args ...interface{})
}

// SyncLogger is a log processing information structure
type SyncLogger struct {
	consoleFileLogger *lumberjack.Logger
	jsonFileLogger    *lumberjack.Logger
	zapLogger         *zap.Logger
}

var Log Logger = &SyncLogger{}

// InitializeLogger initialize console logger and json logger.
func (s *SyncLogger) InitializeLogger() {
	// Set lumberjack - automatically manages log files
	s.consoleFileLogger = s.newLumberJackLogger(config.ConsoleLogFilePath)
	s.jsonFileLogger = s.newLumberJackLogger(config.JsonLogFilePath)

	// Encoder configuration
	consoleEncoderConfig := zapcore.EncoderConfig{
		MessageKey:       "msg",
		LevelKey:         "level",
		TimeKey:          "time",
		CallerKey:        "caller",
		FunctionKey:      zapcore.OmitKey,
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      s.capitalLevelEncoder,
		EncodeTime:       zapcore.TimeEncoderOfLayout("[2006-01-02 15:04:05]"),
		EncodeDuration:   zapcore.SecondsDurationEncoder,
		EncodeCaller:     s.wrapShortCallerEncoder(true),
		ConsoleSeparator: " ",
	}
	jsonEncoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   s.wrapShortCallerEncoder(false),
	}

	// Define console and JSON encoders
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
	jsonEncoder := zapcore.NewJSONEncoder(jsonEncoderConfig)

	// Setup core log writers for console and JSON outputs
	consoleWriter := zapcore.AddSync(s.consoleFileLogger)
	jsonWriter := zapcore.AddSync(s.jsonFileLogger)

	// Creating core
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleWriter, zapcore.InfoLevel),
		zapcore.NewCore(jsonEncoder, jsonWriter, zapcore.InfoLevel),
	)

	// Creating logger with core
	s.zapLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.PanicLevel))
}

// FinalizeLogger At the end of the program, all logs remaining
// in the buffer are written to the file, and open log files are closed.
func (s *SyncLogger) FinalizeLogger() {
	// Flush any buffered log entries
	s.zapLogger.Sync()
	// Close log files
	s.consoleFileLogger.Close()
	s.jsonFileLogger.Close()
}

// newLumberJackLogger create lumberjack logger
//
// Parameters:
//   - logFilePath: log file path
//
// Returns:
//   - *lumberjack.Logger: lumberjack logger
func (s *SyncLogger) newLumberJackLogger(logFilePath string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    config.Conf.MaxLogFileSize,
		MaxBackups: config.Conf.MaxLogFileBackup,
		MaxAge:     config.Conf.MaxLogFileAge,
		Compress:   config.Conf.CompBakLogFile,
	}
}

// capitalLevelEncoder customize zap core CapitalLevelEncoder() method.
// Parameters:
//   - l: log level
//   - enc: array interface
func (s *SyncLogger) capitalLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + l.CapitalString() + "]")
}

// wrapShortCallerEncoder is a wrapping method of zapcore's ShortCallerEncoder method.
//
// Parameters:
//   - isConsole: true(console log), false(json log)
//
// Returns:
//   - func: original method
func (s *SyncLogger) wrapShortCallerEncoder(isConsole bool) func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	return func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		fileIdx := -1
		funcIdx := -1

		if !caller.Defined {
			enc.AppendString(s.putSquareBracketsOnCaller(isConsole, "undefined"))
			return
		}

		// Get file name index
		if fileIdx = strings.LastIndex(caller.File, "/"); fileIdx == -1 {
			enc.AppendString(s.putSquareBracketsOnCaller(isConsole,
				fmt.Sprintf("%s-%s()", caller.FullPath(), caller.Function)))
			return
		}

		// Get function name index
		if funcIdx = strings.LastIndex(caller.Function, "."); funcIdx == -1 {
			enc.AppendString(s.putSquareBracketsOnCaller(isConsole,
				fmt.Sprintf("%s-%s()", caller.FullPath(), caller.Function)))
			return
		}

		// Make caller message and append caller string to log
		enc.AppendString(s.putSquareBracketsOnCaller(isConsole,
			fmt.Sprintf("%s:%d-%s()", caller.File[fileIdx+1:], caller.Line,
				caller.Function[funcIdx+1:])))
	}
}

// putSquareBracketsOnCaller put square brackets on the callers if they are console logs.
//
// Parameters:
//   - isConsole: true(console log), false(json log)
//   - format: caller message
//
// Returns:
//   - string: caller messages with brackets in some cases
func (s *SyncLogger) putSquareBracketsOnCaller(isConsole bool, format string) string {
	if isConsole {
		return "[" + format + "]"
	}
	return format
}

// LogInfo write a log with a log level of INFO.
//
// Parameters:
//   - format: log message
//   - args: variable factor
func (s *SyncLogger) LogInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	s.zapLogger.Info(message)
}

// LogWarn write a log with a log level of WARN.
//
// Parameters:
//   - format: log message
//   - args: variable factor
func (s *SyncLogger) LogWarn(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	s.zapLogger.Warn(message)
}

// LogError write a log with a log level of ERROR.
//
// Parameters:
//   - format: log message
//   - args: variable factor
func (s *SyncLogger) LogError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	s.zapLogger.Error(message)
}

// LogDebug write a log with a log level of DEBUG.
//
// Parameters:
//   - format: log message
//   - args: variable factor
func (s *SyncLogger) LogDebug(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	s.zapLogger.Debug(message)
}

// LogPanic write a log with a log level of PANIC.
// The logger then panics, even if logging at PanicLevel is disabled.
//
// Parameters:
//   - format: log message
//   - args: variable factor
func (s *SyncLogger) LogPanic(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	s.zapLogger.Panic(message)
}

// LogFatal write a log with a log level of FATAL.
// The logger then calls os.Exit(1), even if logging at FatalLevel is
// disabled.
//
// Parameters:
//   - format: log message
//   - args: variable factor
func (s *SyncLogger) LogFatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	s.zapLogger.Fatal(message)
}
