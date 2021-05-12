package request

import (
	"net/http"
	"time"
)

var client = &http.Client{
	Timeout:   time.Second * 3,
	Transport: tr,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}
