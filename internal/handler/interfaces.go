package handler

type Logger interface {
	Warnw(message string, params ...any)
	Errorw(message string, params ...any)
}
