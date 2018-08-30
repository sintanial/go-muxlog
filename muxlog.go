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
	TokenRequestBytes    = "%reqb%"
	TokenRequestHeaders  = "%reqh%"
	TokenUserAgent       = "%ua%"
	TokenResponseCode    = "%rescode%"
	TokenResponseStatus  = "%resst%"
	TokenResponseBytes   = "%resb%"
	TokenResponseHeaders = "%resh%"
)

var DefaultFormat = `%raddr% > |%mtd%| %uri% %reqb% < %rescode%(%resst%) %resb%; "%ua%"`

type LoggingFunc func(rw *ResponseWriter, msg string, err error)

type ServeMux struct {
	format string
	logger LoggingFunc
	mux    *http.ServeMux
}

func NewDefault() *ServeMux {
	return New(http.NewServeMux())
}

func New(mux *http.ServeMux) *ServeMux {
	return NewWithLogger(mux, nil)
}

func NewWithLogger(mux *http.ServeMux, log LoggingFunc) *ServeMux {
	return &ServeMux{
		format: DefaultFormat,
		logger: log,
		mux:    mux,
	}
}

type ResponseWriter struct {
	http.ResponseWriter

	ResponseStatusCode int
	WritedBytes        int
	IsSended           bool
}

func (self *ResponseWriter) WriteHeader(status int) {
	self.ResponseStatusCode = status
	self.IsSended = true
	self.ResponseWriter.WriteHeader(status)
}

func (self *ResponseWriter) Write(b []byte) (int, error) {
	n, err := self.ResponseWriter.Write(b)
	self.WritedBytes = n
	self.IsSended = true
	return n, err
}

func (self *ServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request) error) {
	self.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		rw := &ResponseWriter{ResponseWriter: w}
		self.log(rw, r, handler(rw, r))
	})
}

func (self *ServeMux) log(rw *ResponseWriter, r *http.Request, err error) {
	if self.logger == nil || self.format == "" {
		return
	}

	status := http.StatusOK
	if rw.ResponseStatusCode != 0 {
		status = rw.ResponseStatusCode
	}
	userAgent := r.UserAgent()
	r.Header.Del("User-Agent")

	var msg string

	msg = self.format
	msg = strings.Replace(msg, TokenRemoteAddr, r.RemoteAddr, -1)
	msg = strings.Replace(msg, TokenMethod, r.Method, -1)
	msg = strings.Replace(msg, TokenRequestURI, r.RequestURI, -1)
	msg = strings.Replace(msg, TokenRequestBytes, strconv.Itoa(int(r.ContentLength)), -1)
	msg = strings.Replace(msg, TokenRequestHeaders, fmt.Sprintf("%+v", r.Header), -1)
	msg = strings.Replace(msg, TokenUserAgent, userAgent, -1)
	msg = strings.Replace(msg, TokenResponseCode, strconv.Itoa(status), -1)
	msg = strings.Replace(msg, TokenResponseStatus, http.StatusText(status), -1)
	msg = strings.Replace(msg, TokenResponseBytes, strconv.Itoa(rw.WritedBytes), -1)
	msg = strings.Replace(msg, TokenResponseHeaders, fmt.Sprintf("%+v", rw.Header()), -1)

	self.logger(rw, msg, err)
}

// set logger
func (self *ServeMux) SetLogger(l LoggingFunc) {
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
