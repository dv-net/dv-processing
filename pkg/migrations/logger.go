package migrations

type initLogger interface {
	Infof(format string, args ...interface{})
}

type logger struct {
	log initLogger
}

func newLogger(log initLogger) *logger {
	return &logger{
		log: log,
	}
}

// Printf logs a message at the info level.
func (l *logger) Printf(format string, args ...interface{}) {
	l.log.Infof(format, args...)
}

// Verbose
func (l logger) Verbose() bool { return false }
