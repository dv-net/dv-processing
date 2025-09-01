package workflow

type logger interface {
	Debugw(format string, args ...any)
	Debugf(format string, args ...any)
	Infow(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Warnw(format string, args ...any)
	Errorw(format string, args ...any)
	Errorf(format string, args ...any)
}
