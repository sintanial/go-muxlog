package muxlog

import (
	"net/http"
	"strings"
	"strconv"
	"fmt"
	"bytes"
	"io/ioutil"
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

type LoggingFunc func(w *ResponseRecorder, r *RequestRecorder, msg string, err error)

type RequestRecorder struct {
	*http.Request
	Body *bytes.Buffer
}

type ResponseRecorder struct {
	w http.ResponseWriter

	StatusCode int
	BodyBytes  int
	Body       *bytes.Buffer

	isResponded bool

	isNeedLogBody bool
}

func (self *ResponseRecorder) Header() http.Header {
	return self.w.Header()
}

func (self *ResponseRecorder) WriteHeader(status int) {
	if self.isResponded {
		return
	}

	self.StatusCode = status
	self.isResponded = true

	self.w.WriteHeader(status)
}

func (self *ResponseRecorder) Write(b []byte) (int, error) {
	if self.isResponded {
		return 0, nil
	}

	if self.Body == nil && self.isNeedLogBody {
		self.Body = bytes.NewBuffer(b)
	}

	n, err := self.w.Write(b)
	self.BodyBytes = n

	self.StatusCode = http.StatusOK
	self.isResponded = true

	return n, err
}

type ServeMux struct {
	format string
	logger LoggingFunc
	mux    *http.ServeMux

	isNeedLogRequestBody  bool
	isNeedLogResponseBody bool
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

type readerWithErr struct {
	err error
}

func (self readerWithErr) Read(b []byte) (int, error) {
	return 0, self.err
}

func (self *ServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	self.Handle(pattern, http.HandlerFunc(handler))
}

func (self *ServeMux) Handle(pattern string, handler http.Handler) {
	self.HandleFuncError(pattern, func(w http.ResponseWriter, r *http.Request) error {
		handler.ServeHTTP(w, r)
		return nil
	})
}

func (self *ServeMux) HandleFuncError(pattern string, handler func(http.ResponseWriter, *http.Request) error) {
	self.mux.Handle(pattern, self.WrapError(handler))
}

func (self *ServeMux) Wrap(handler func(http.ResponseWriter, *http.Request)) http.Handler {
	return self.WrapError(func(w http.ResponseWriter, r *http.Request) error {
		handler(w, r)
		return nil
	})
}

func (self *ServeMux) WrapError(handler func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resr := &ResponseRecorder{
			w: w,

			isNeedLogBody: self.isNeedLogResponseBody,
		}

		reqr := &RequestRecorder{
			Request: r,
		}

		if self.isNeedLogRequestBody {
			data, err := ioutil.ReadAll(r.Body)
			if err == nil {
				r.Body = ioutil.NopCloser(bytes.NewReader(data))
				reqr.Body = bytes.NewBuffer(data)
			} else {
				r.Body = ioutil.NopCloser(readerWithErr{err})
			}
		}

		err := handler(resr, r)
		self.log(resr, reqr, err)
	})
}

func (self *ServeMux) log(w *ResponseRecorder, r *RequestRecorder, err error) {
	if self.logger == nil || self.format == "" {
		return
	}

	status := http.StatusOK
	if w.StatusCode != 0 {
		status = w.StatusCode
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
	msg = strings.Replace(msg, TokenResponseBytes, strconv.Itoa(w.BodyBytes), -1)
	msg = strings.Replace(msg, TokenResponseHeaders, fmt.Sprintf("%+v", w.Header()), -1)

	self.logger(w, r, msg, err)
}

// set logger
func (self *ServeMux) SetLogger(l LoggingFunc) {
	self.logger = l
}

// set logging format
// todo: prepare result format, needed for improve performance (replace strings.Replace to Tokenizer)
func (self *ServeMux) SetFormat(f string) {
	self.format = f
}

func (self *ServeMux) SetLogRequestBody(b bool) {
	self.isNeedLogRequestBody = true
}

func (self *ServeMux) SetLogResponse(b bool) {
	self.isNeedLogResponseBody = true
}

func (self *ServeMux) Mux() *http.ServeMux {
	return self.mux
}
