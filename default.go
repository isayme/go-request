package request

import (
	"context"
	"net/http"
)

var defaultRequestOpts = New()

// OptionFunc update option function
type OptionFunc func(*Option)

// WithOption update default option
func WithOption(fn OptionFunc) {
	fn(defaultRequestOpts)
}

// Request default request instance
var Request = func(ctx context.Context, method, url string, header http.Header, body interface{}, out interface{}) error {
	_, err := defaultRequestOpts.Request(ctx, method, url, header, body, out)
	return err
}

// RequestWithResponse return response
var RequestWithResponse = defaultRequestOpts.Request
