package listen

import (
	"os"

	"github.com/sirupsen/logrus"
)

func newLogger(disableTimestamp bool, logLevelString string) (logrus.FieldLogger, error) {
	logLevel, err := logrus.ParseLevel(logLevelString)
	if err != nil {
		return nil, err
	}

	return &logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.TextFormatter{
			DisableTimestamp: disableTimestamp,
		},
		Level: logLevel,
	}, nil
}
