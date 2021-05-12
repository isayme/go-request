package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/isayme/go-logger"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
)

type Option struct {
	Client          *http.Client
	UserAgentPrefix string
	MaxRetry        int
	RetryDelay      func(int) time.Duration
	ShouldRetryFn   func(error) bool
}

func New() *Option {
	return &Option{
		Client:   client,
		MaxRetry: 3,
		RetryDelay: func(retry int) time.Duration {
			return time.Millisecond * 100 * time.Duration(retry)
		},
		ShouldRetryFn:   shouldRetry,
		UserAgentPrefix: "",
	}
}

// Request 对外发起请求
func (opts Option) Request(ctx context.Context, method, url string, header http.Header, body interface{}, out interface{}) (resp *http.Response, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "request.Request")
	defer span.Finish()

	// 默认 json
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}

	if header.Get("Accept") == "" {
		header.Set("Accept", "application/json")
	}

	// 默认UserAgent
	if header.Get("User-Agent") == "" {
		userAgent := UserAgent
		if opts.UserAgentPrefix != "" {
			userAgent = fmt.Sprintf("%s %s", opts.UserAgentPrefix, UserAgent)
		}
		header.Set("User-Agent", userAgent)
	}

	xRequestID := header.Get("X-Request-Id")
	if xRequestID == "" {
		xRequestID = uuid.NewV4().String()
		header.Set("X-Request-Id", xRequestID)
	}

	// 注入 trace header
	if span != nil {
		opentracing.GlobalTracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(header))
	}

	bs := []byte{}

	if body != nil {
		bs, err = json.Marshal(body)
		if err != nil {
			logger.Errorw("jsonMarshal failed", "method", method, "url", url, "err", err.Error())
			return
		}
	}

	var delay time.Duration

	start := time.Now()

	retry := 0
	for ; ; retry++ {
		clientDoSpan, _ := opentracing.StartSpanFromContext(ctx, "request.Request.clientDo")
		var req *http.Request
		req, err = http.NewRequest(method, url, bytes.NewReader(bs))
		if err != nil {
			logger.Errorw("http.NewRequest failed", "method", method, "url", url, "err", err.Error())
			return
		}

		req.Header = header
		if retry > 0 {
			req.Header.Set("X-Retry", strconv.Itoa(retry))
		}

		now := time.Now()
		resp, err = opts.Client.Do(req)
		clientDoSpan.Finish()

		if err != nil {
			if resp != nil {
				resp.Body.Close()
			}

			// 重试次数未满 且 可以重试
			if retry < opts.MaxRetry && opts.ShouldRetryFn(err) {
				logger.Warnw("client.Do failed",
					"method", method,
					"url", url,
					"retry", retry,
					"requestId", xRequestID,
					"start", now.String(),
					"duration", toMs(time.Since(now)),
					"err", err.Error(),
				)

				delay = opts.RetryDelay(retry + 1)
				time.Sleep(delay)
				continue
			}

			// 不可重试, 退出
			logger.Errorw("client.Do failed",
				"method", method,
				"url", url,
				"retry", retry,
				"requestId", xRequestID,
				"start", now.String(),
				"duration", toMs(time.Since(now)),
				"err", err.Error(),
			)

			return nil, fmt.Errorf("request fail: %w", err)
		}
		// 正常响应
		break
	}

	defer resp.Body.Close()

	readBodySpan, _ := opentracing.StartSpanFromContext(ctx, "request.Request.ReadRespBody")
	respBody, err := ioutil.ReadAll(resp.Body)
	readBodySpan.Finish()

	logger.Debugw("request",
		"method", method,
		"url", url,
		"retry", retry,
		"start", start.String(),
		"duration", toMs(time.Since(start)),
		"respCode", resp.StatusCode,
		"respBody", string(respBody),
	)
	if err != nil {
		return
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request(%s) fail, status: %d, requestId: %s, body: %s", url, resp.StatusCode, string(respBody), xRequestID)
	}

	if out != nil {
		err = json.Unmarshal(respBody, out)
		if err != nil {
			logger.Errorw("jsonUnmarshal failed",
				"method", method,
				"url", url,
				"respBody", string(respBody),
				"err", err.Error(),
			)
			return
		}
	}

	return
}
