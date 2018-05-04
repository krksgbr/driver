package logging

import (
	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
)

type SystemHook struct {
	logger    service.Logger
	formatter *logrus.TextFormatter
}

func NewSystemHook(systemLogger service.Logger) *SystemHook {
	return &SystemHook{
		logger:    systemLogger,
		formatter: &logrus.TextFormatter{DisableTimestamp: true},
	}
}

func (hook SystemHook) Fire(entry *logrus.Entry) error {
	bytes, err := hook.formatter.Format(entry)
	if err != nil {
		return err
	}
	line := string(bytes)

	switch entry.Level {
	case logrus.PanicLevel:
		return hook.logger.Error(line)
	case logrus.FatalLevel:
		return hook.logger.Error(line)
	case logrus.ErrorLevel:
		return hook.logger.Error(line)
	case logrus.WarnLevel:
		return hook.logger.Warning(line)
	case logrus.InfoLevel:
		return hook.logger.Info(line)
	default:
		return nil
	}
}

func (hook SystemHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
}
