package log

import (
	logs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	thinkconfig "github.com/watsonhaw5566/think-core/config"
	"io"
	"os"
	"path/filepath"
	"time"
)

var log *logrus.Logger       // 控制台输入日志
var logToFile *logrus.Logger // 文件写入日志
var loggerFile string        // 日志路径

// 初始化日志路径
func init() {
	loggerFile = filepath.Join(thinkconfig.Config.Log.Path, thinkconfig.Config.Log.Name)
}

// Log 日志方法
func Log() *logrus.Logger {
	if thinkconfig.Config.Log.Model == "file" {
		// 文件输出
		return logFile()
	} else {
		// 控制台输出
		if log == nil {
			log = logrus.New()
			log.Out = os.Stdout
			log.Formatter = &logrus.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05"}
			log.SetLevel(logrus.DebugLevel)
		}
		return log
	}
}

// 文件写入
func logFile() *logrus.Logger {
	if logToFile == nil {
		logToFile = logrus.New()
		logToFile.SetLevel(logrus.DebugLevel)
		logToFile.Out = io.Discard

		logWriter, _ := logs.New(
			loggerFile+"_%Y%m%d.log",
			logs.WithMaxAge(time.Duration(thinkconfig.Config.Log.MaxAge)*24*time.Hour),
			logs.WithRotationTime(24*time.Hour),
		)

		errorLogWriter, _ := logs.New(
			loggerFile+"_error_%Y%m%d.log",
			logs.WithMaxAge(time.Duration(thinkconfig.Config.Log.MaxAge)*24*time.Hour),
			logs.WithRotationTime(24*time.Hour),
		)

		writeMap := lfshook.WriterMap{
			logrus.InfoLevel:  logWriter,
			logrus.FatalLevel: logWriter,
			logrus.DebugLevel: logWriter,
			logrus.WarnLevel:  logWriter,
			logrus.ErrorLevel: errorLogWriter,
			logrus.PanicLevel: logWriter,
		}

		lfHook := lfshook.NewHook(writeMap, &logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
		logToFile.AddHook(lfHook)
	}
	return logToFile
}
