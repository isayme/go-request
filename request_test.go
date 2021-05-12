package request

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequest(t *testing.T) {
	require := require.New(t)

	ctx := context.Background()
	method := http.MethodGet
	url := "http://httpbin.org/anything"

	err := Request(ctx, method, url, http.Header{}, nil, nil)
	require.Nil(err)
}
