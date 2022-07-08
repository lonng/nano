package logger

import "github.com/cute-angelia/go-utils/components/loggerV3"

type logger struct {
}

func NewLogger() *logger {
	return &logger{}
}

func (l logger) Println(v ...interface{}) {
	loggerV3.GetLogger().Info().Msgf("%v", v)
}

func (l logger) Fatal(v ...interface{}) {
	loggerV3.GetLogger().Fatal().Msgf("%v", v)
}

func (l logger) Fatalf(format string, v ...interface{}) {
	loggerV3.GetLogger().Fatal().Msgf(format, v)
}
