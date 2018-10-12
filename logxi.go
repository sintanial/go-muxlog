package muxlog

import (
	"bitbucket.org/MountAim/go-lerry"
	"github.com/ansel1/merry"
	"github.com/mgutz/logxi/v1"
)

func LogxiLogingFunc(l log.Logger) LoggingFunc {
	return func(resr *ResponseRecorder, reqr *RequestRecorder, msg string, err error) {
		if err != nil {
			level, message, args := lerry.Prepare(err)
			newargs := append([]interface{}{}, "reason", message)
			newargs = append(newargs, args...)

			if err != nil {
				resr.WriteHeader(merry.HTTPCode(err))
			}

			l.Log(level, msg, newargs)
		} else if resr.StatusCode >= 500 && resr.StatusCode <= 599 {
			l.Warn(msg)
		} else if resr.StatusCode >= 400 && resr.StatusCode <= 499 {
			l.Info(msg)
		} else {
			l.Debug(msg)
		}
	}

}
