package main

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	w       http.ResponseWriter
	Code    int         `json:"code"` // Add this fiel
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    interface{} `json:"meta"`
}

func NewResponse(w http.ResponseWriter, code int, message string) *Response {
	return &Response{
		w:       w,
		Code:    code,
		Message: message,
	}
}

func (r *Response) WithData(data interface{}) *Response {
	r.Data = data
	return r
}

func (r *Response) WithMeta(meta interface{}) *Response {
	r.Meta = meta
	return r
}

func (r *Response) Error(w http.ResponseWriter, code int, message string) {
	r.Code = code
	r.Message = message
	r.JSON()
}

func (r *Response) Success(w http.ResponseWriter, message string, data interface{}) {
	r.Code = http.StatusOK
	r.Message = message
	r.Data = data
	r.JSON()
}

func (r *Response) JSON() {
	r.w.Header().Set("Content-Type", "application/json")
	r.w.WriteHeader(r.Code)
	json.NewEncoder(r.w).Encode(r)
}
