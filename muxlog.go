package muxlog

import (
	"net/http"
	"strings"
	"strconv"
	"fmt"
)

const (
	TokenRemoteAddr      = "%raddr%"
	TokenMethod          = "%mtd%"
	TokenRequestURI      = "%uri%"
	TokenRequestHeaders  = "%reqh%"
	TokenUserAgent       = "%ua%"
	TokenResponseCode    = "%rescode%"
	TokenResponseStatus  = "%resst%"
	TokenResponseBytes   = "%resb%"
	TokenResponseHeaders = "%resh%"
)

var DefaultFormat = `%raddr% > |%mtd%| %uri% %reqh% %ua% < %rescode%(%resst%) %resb% %resh%`

type Logger interface {
	Log(msg string, err error)
}

type ServeMux struct {
	format string
	logger Logger
	mux    *http.ServeMux
}

func NewDefault() *ServeMux {
	return New(http.NewServeMux())
}

func New(mux *http.ServeMux) *ServeMux {
	return NewLogger(mux, nil)
}

func NewLogger(mux *http.ServeMux, log Logger) *ServeMux {
	return &ServeMux{
		format: DefaultFormat,
		logger: log,
		mux:    mux,
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (self *responseWriter) WriteHeader(status int) {
	self.status = status
	self.ResponseWriter.WriteHeader(status)
}

func (self *responseWriter) Write(b []byte) (int, error) {
	n, err := self.ResponseWriter.Write(b)
	self.bytes = n

	return n, err
}

func (self *ServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request) error) {
	self.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w}
		self.log(rw, r, handler(rw, r))
	})
}

func (self *ServeMux) log(rw *responseWriter, r *http.Request, err error) {
	if self.logger == nil || self.format == "" {
		return
	}

	var msg string

	msg = self.format
	msg = strings.Replace(self.format, TokenRemoteAddr, r.RemoteAddr, -1)
	msg = strings.Replace(self.format, TokenMethod, r.Method, -1)
	msg = strings.Replace(self.format, TokenRequestURI, r.RequestURI, -1)
	msg = strings.Replace(self.format, TokenRequestHeaders, fmt.Sprintf("%+v", r.Header), -1)
	msg = strings.Replace(self.format, TokenUserAgent, r.UserAgent(), -1)
	msg = strings.Replace(self.format, TokenResponseCode, strconv.Itoa(rw.status), -1)
	msg = strings.Replace(self.format, TokenResponseStatus, http.StatusText(rw.status), -1)
	msg = strings.Replace(self.format, TokenResponseBytes, strconv.Itoa(rw.bytes), -1)
	msg = strings.Replace(self.format, TokenResponseHeaders, fmt.Sprintf("%+v", rw.Header()), -1)

	self.logger.Log(msg, err)
}

// set logger
func (self *ServeMux) SetLogger(l Logger) {
	self.logger = l
}

// set logging format
// todo: prepare result format, needed for improve performan (replace strings.Replace to Tokenizer)
func (self *ServeMux) SetFormat(f string) {
	self.format = f
}

func (self *ServeMux) Mux() *http.ServeMux {
	return self.mux
}
