package mqtt

import (
	"github.com/sirupsen/logrus"
)

// Logger is a wrapper around the logrus logger for the MQTT client.
// This allows the same logging as the rest of the app. It also supports the log
// level set in the configuration.
type Logger struct {
	level  logrus.Level
	logger logrus.FieldLogger
}

func NewLogger(logger logrus.FieldLogger, level logrus.Level) *Logger {
	return &Logger{
		level:  level,
		logger: logger,
	}
}

func (l *Logger) Println(v ...interface{}) {
	switch l.level {
	case logrus.PanicLevel:
		l.logger.Panicln(v)
	case logrus.FatalLevel:
		l.logger.Fatalln(v)
	case logrus.ErrorLevel:
		l.logger.Errorln(v)
	case logrus.WarnLevel:
		l.logger.Warnln(v)
	case logrus.InfoLevel:
		l.logger.Infoln(v)
	case logrus.DebugLevel:
		l.logger.Debugln(v)
	}
}
func (l *Logger) Printf(format string, v ...interface{}) {
	switch l.level {
	case logrus.PanicLevel:
		l.logger.Panicf(format, v)
	case logrus.FatalLevel:
		l.logger.Fatalf(format, v)
	case logrus.ErrorLevel:
		l.logger.Errorf(format, v)
	case logrus.WarnLevel:
		l.logger.Warnf(format, v)
	case logrus.InfoLevel:
		l.logger.Infof(format, v)
	case logrus.DebugLevel:
		l.logger.Debugf(format, v)
	}
}
