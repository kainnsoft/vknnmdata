package logging

import (
	"io"
	logging "log"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// ****************************************************************************
// логгер для вывода логов в файл логов (или в stdOut)
type infoLog struct {
	out *logrus.Logger
	mx  *sync.Mutex
}

func newInfoLog(outPath string, mtx *sync.Mutex) *infoLog {
	infoLogFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
	if err != nil {
		logging.Fatalf("error opening infoLogFile: %v", err)
	}

	inflog := logrus.New()
	inflog.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	inflog.SetReportCaller(false) // func name

	mw := io.MultiWriter(os.Stdout, infoLogFile)
	inflog.SetOutput(mw)

	infolog := infoLog{out: inflog, mx: mtx}

	return &infolog
}

func (l *infoLog) infof(format string, v ...interface{}) {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.out.Infof(format, v...)
}

// ----------------------
// логгер для вывода логов ошибок в файл логов ошибок (или в stdErr)
type errorLog struct {
	out *logrus.Logger
	mx  *sync.Mutex
}

func newErrorLog(outPath string, mtx *sync.Mutex) *errorLog {
	errorLogFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
	if err != nil {
		logging.Fatalf("error opening errorLogFile: %v", err)
	}

	elog := logrus.New()
	elog.SetFormatter(&logrus.TextFormatter{ // log.Formatter = &logrus.JSONFormatter{}
		DisableColors: true,
		FullTimestamp: true,
	})

	elog.SetReportCaller(true) // func name

	mw := io.MultiWriter(os.Stdout, errorLogFile)
	elog.SetOutput(mw)

	errlog := errorLog{out: elog, mx: mtx}

	return &errlog
}

func (l *errorLog) warnf(format string, v ...interface{}) {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.out.Warnf(format, v...)
}

func (l *errorLog) errorf(format string, v ...interface{}) {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.out.Errorf(format, v...)
}

func (l *errorLog) fatalf(format string, v ...interface{}) {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.out.Fatalf(format, v...)
}

// ----------------------
// логгер для вывода логов в спец файл для бухгалтерии
type buhLog struct {
	out *logrus.Logger
	mx  *sync.Mutex
}

func newBuhLog(outPath string, mtx *sync.Mutex) *buhLog {
	buhLogFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
	if err != nil {
		logging.Fatalf("error opening buhLogFile: %v", err)
	}

	blog := logrus.New()
	blog.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: false,
	})

	blog.SetReportCaller(false) // func name

	mw := io.MultiWriter(os.Stdout, buhLogFile)
	blog.SetOutput(mw)

	buhlog := buhLog{out: blog, mx: mtx}

	return &buhlog
}

func (l *buhLog) infof(format string, v ...interface{}) {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.out.Infof(format, v...)
}

// ----------------------
// главный логгер. Запускается при старте системы
type MDLogger struct {
	inflog *infoLog
	errlog *errorLog
	buhlog *buhLog
}

func NewMDLogger(infoPath, errPath, buhPath string) *MDLogger {
	var mtx sync.Mutex
	infolog := newInfoLog(infoPath, &mtx)
	errlog := newErrorLog(errPath, &mtx)
	buhlog := newBuhLog(buhPath, &mtx)

	return &MDLogger{inflog: infolog,
		errlog: errlog,
		buhlog: buhlog}
}

func (mdl *MDLogger) Infof(format string, v ...interface{}) {
	mdl.inflog.infof(format, v...)
}

func (mdl *MDLogger) Warnf(format string, v ...interface{}) {
	mdl.errlog.warnf(format, v...)
}

func (mdl *MDLogger) Errorf(format string, v ...interface{}) {
	mdl.errlog.errorf(format, v...)
}

func (mdl *MDLogger) Fatalf(format string, v ...interface{}) {
	mdl.errlog.fatalf(format, v...)
}

func (mdl *MDLogger) BuhInfof(format string, v ...interface{}) {
	mdl.buhlog.infof(format, v...)
}
