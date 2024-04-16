package cmd

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path"
	"time"
)

var lumberjackLogger *lumberjack.Logger

func InitLogger() {
	lumberjackLogger := &lumberjack.Logger{
		Filename:   path.Join(AppConfig.LogFilePath, "/FileCleanup/log/FileCleanup.log"),
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	// Fork writing into two outputs
	multiWriter := io.MultiWriter(os.Stderr, lumberjackLogger)

	logFormatter := new(log.TextFormatter)
	logFormatter.TimestampFormat = time.DateTime // or RFC3339
	logFormatter.FullTimestamp = true

	log.SetFormatter(logFormatter)
	log.SetOutput(multiWriter)
}
