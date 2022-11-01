package logging

import (
	"io"
	logging "log"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	log, logBuch, logBuchFull *logrus.Logger
	FBuchName                 string
)

var lock = sync.Mutex{}

func init() {
	mainLogFile, err := os.OpenFile("logs/md-info.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
	if err != nil {
		logging.Fatalf("error opening file: %v", err)
	}

	log = logrus.New()
	//log.Formatter = &logrus.JSONFormatter{}

	log.SetReportCaller(false)

	mw := io.MultiWriter(os.Stdout, mainLogFile)
	log.SetOutput(mw)

	{
		//*************************************************************************************
		// for Gruschenko еженедельный. Каждую пятницу отправляется и очищается.
		fBuch, err := os.OpenFile("logs/Электронные почты сотрудников.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
		FBuchName = fBuch.Name()
		if err != nil {
			logging.Fatalf("error opening file: %v", err)
		}
		logBuch = logrus.New()
		logBuch.SetReportCaller(false)
		mwBuch := io.MultiWriter(os.Stdout, fBuch)
		logBuch.SetOutput(mwBuch)

		//*************************************************************************************
		// for Gruschenko max итоговый за все перриоды (нреочищаемый)
		fBuchFull, err := os.OpenFile("logs/mdBuch-info.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
		if err != nil {
			logging.Fatalf("error opening file: %v", err)
		}
		logBuchFull = logrus.New()
		logBuch.SetReportCaller(false)
		mwBuchFull := io.MultiWriter(os.Stdout, fBuchFull)
		logBuchFull.SetOutput(mwBuchFull)
	}
}

// Info ...
func Info(format string, v ...interface{}) {
	lock.Lock()
	log.Infof(format, v...)
	lock.Unlock()
}

// for Gruschenko
func InfoBuch(format string, v ...interface{}) {
	lock.Lock()
	logBuch.Infof(format, v...)
	lock.Unlock()
}
func InfoBuchFull(format string, v ...interface{}) {
	lock.Lock()
	logBuchFull.Infof(format, v...)
	lock.Unlock()
}

// Warn ...
func Warn(format string, v ...interface{}) {
	lock.Lock()
	log.Warnf(format, v...)
	lock.Unlock()
}

// Error ...
func Error(format string, v ...interface{}) {
	lock.Lock()
	log.Errorf(format, v...)
	lock.Unlock()
}

var (

	// ConfigError ...
	ConfigError = "%v type=config.error"

	// HTTPError ...
	HTTPError = "%v type=http.error"

	// HTTPWarn ...
	HTTPWarn = "%v type=http.warn"

	// HTTPInfo ...
	HTTPInfo = "%v type=http.info"
)
