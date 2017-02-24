package server

import (
	"strings"

	"github.com/sirupsen/logrus"
)

type FTPLogger struct{}

func (logger *FTPLogger) Print(sessionId string, message interface{}) {
	logrus.WithFields(logrus.Fields{"Session": sessionId, "Message": message}).Debug("FTP:", message)
}
func (logger *FTPLogger) PrintCommand(sessionId string, command string, params string) {
	logrus.WithFields(logrus.Fields{"Session": sessionId, "Command": command, "Parameters": params}).Debugf("FTP: %s(%s)", command, params)
}
func (logger *FTPLogger) PrintResponse(sessionId string, code int, message string) {
	logrus.WithFields(logrus.Fields{"Session": sessionId, "Code": code, "Response": message}).Debugf("Response with %q and code %d", message, code)

}
func (logger *FTPLogger) Printf(sessionId string, format string, v ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	logrus.WithFields(logrus.Fields{"Session": sessionId}).Debugf(format, v...)
}
