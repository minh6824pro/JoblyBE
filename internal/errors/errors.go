package errors

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Error struct {
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	HTTPCode int            `json:"http_code"`
	Err      error          `json:"-"`
	Meta     map[string]any `json:"meta,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(code string, message string, hc int, e error) error {
	return &Error{
		Code:     code,
		Message:  message,
		Err:      e,
		HTTPCode: hc,
	}
}

func NewErrorWithMeta(code string, message string, hc int, e error, meta map[string]any) error {
	return &Error{
		Code:     code,
		Message:  message,
		Err:      e,
		HTTPCode: hc,
		Meta:     meta,
	}
}

func ParseError(err error) *Error {
	if e, ok := err.(*Error); ok {
		return e
	}

	return &Error{
		Code:     INTERNAL_ERROR,
		Message:  err.Error(),
		HTTPCode: http.StatusInternalServerError,
		Err:      err,
	}
}

func WriteError(c *gin.Context, err error) {
	e := ParseError(err)
	res := gin.H{
		"error":   e.Code,
		"message": e.Message,
	}
	if e.Meta != nil {
		for k, v := range e.Meta {
			res[k] = v
		}
	}
	c.JSON(e.HTTPCode, res)
}
