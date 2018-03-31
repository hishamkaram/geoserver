package geoserver

import (
	"github.com/sirupsen/logrus"
)

//GetLogger return logger
func GetLogger() (logger *logrus.Logger) {
	logger = logrus.New()
	Formatter := new(logrus.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	logger.Formatter = Formatter
	return
}
