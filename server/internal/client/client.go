package client

import (
	"context"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/fshttp"
	"github.com/rclone/rclone/lib/rest"
)

// Client is a type alias for rest.Client
type Client = rest.Client

// Option modifies the rclone ConfigInfo to configure the underlying fshttp client
type Option func(*fs.ConfigInfo)

// New creates a new rest.Client backed by a robust fshttp.Client
// It configures the client by injecting a modified fs.ConfigInfo into the context.
func New(ctx context.Context, baseURL string, opts ...Option) *Client {
	// AddConfig creates a shallow copy of the current config (or default)
	// and puts it into the returned context. We get a pointer to the mutable copy.
	ctx, ci := fs.AddConfig(ctx)

	// Apply options to the config copy
	for _, opt := range opts {
		opt(ci)
	}

	baseClient := fshttp.NewClient(ctx)

	// Create the rest client
	rc := rest.NewClient(baseClient)
	if baseURL != "" {
		rc.SetRoot(baseURL)
	}

	return rc
}

// WithTimeout sets the response header timeout.
// This is the total time to wait for a response header after sending the request.
func WithTimeout(d time.Duration) Option {
	return func(ci *fs.ConfigInfo) {
		ci.Timeout = fs.Duration(d)
	}
}

// WithConnectTimeout sets the connection timeout.
// This is the time to wait for a TCP connection to be established.
func WithConnectTimeout(d time.Duration) Option {
	return func(ci *fs.ConfigInfo) {
		ci.ConnectTimeout = fs.Duration(d)
	}
}

// WithExpectContinueTimeout sets the timeout for Expect: 100-continue.
func WithExpectContinueTimeout(d time.Duration) Option {
	return func(ci *fs.ConfigInfo) {
		ci.ExpectContinueTimeout = fs.Duration(d)
	}
}

// WithProxy sets a specific proxy URL (e.g. http://..., socks5://...).
func WithProxy(proxyURL string) Option {
	return func(ci *fs.ConfigInfo) {
		ci.Proxy = proxyURL
	}
}

// WithUserAgent sets the User-Agent header for all requests.
func WithUserAgent(ua string) Option {
	return func(ci *fs.ConfigInfo) {
		ci.UserAgent = ua
	}
}

// WithInsecureSkipVerify controls SSL verification.
// Set to true to disable certificate verification (insecure).
func WithInsecureSkipVerify(skip bool) Option {
	return func(ci *fs.ConfigInfo) {
		ci.InsecureSkipVerify = skip
	}
}

// WithHeader adds a global header to all requests.
// Can be called multiple times to add multiple headers.
func WithHeader(key, value string) Option {
	return func(ci *fs.ConfigInfo) {
		ci.Headers = append(ci.Headers, &fs.HTTPOption{
			Key:   key,
			Value: value,
		})
	}
}

// WithDump enables debug logging for HTTP requests/responses.
// headers: log HTTP headers.
// bodies: log HTTP bodies.
// requests: log HTTP requests.
// responses: log HTTP responses.
// auth: include auth headers in dumps (careful with secrets).
func WithDump(headers, bodies, requests, responses, auth bool) Option {
	return func(ci *fs.ConfigInfo) {
		var flags fs.DumpFlags
		if headers {
			flags |= fs.DumpHeaders
		}
		if bodies {
			flags |= fs.DumpBodies
		}
		if requests {
			flags |= fs.DumpRequests
		}
		if responses {
			flags |= fs.DumpResponses
		}
		if auth {
			flags |= fs.DumpAuth
		}
		ci.Dump = flags
	}
}

// WithDisableHTTP2 forces the client to use HTTP/1.1.
func WithDisableHTTP2(disable bool) Option {
	return func(ci *fs.ConfigInfo) {
		ci.DisableHTTP2 = disable
	}
}

// WithDisableKeepAlives disables HTTP keep-alives (connection pooling).
func WithDisableKeepAlives(disable bool) Option {
	return func(ci *fs.ConfigInfo) {
		ci.DisableHTTPKeepAlives = disable
	}
}

// WithNoGzip disables transparent Gzip compression/decompression.
func WithNoGzip(disable bool) Option {
	return func(ci *fs.ConfigInfo) {
		ci.NoGzip = disable
	}
}

// WithCookieJar enables the session cookie jar.
func WithCookieJar(enable bool) Option {
	return func(ci *fs.ConfigInfo) {
		ci.Cookie = enable
	}
}

// WithCustomCA adds custom CA certificates from files.
func WithCustomCA(caFiles ...string) Option {
	return func(ci *fs.ConfigInfo) {
		ci.CaCert = append(ci.CaCert, caFiles...)
	}
}

// WithClientCert configures client-side TLS certificate for mutual authentication.
// certFile: path to PEM encoded certificate.
// keyFile: path to PEM encoded key.
// password: password for the key (optional).
func WithClientCert(certFile, keyFile, password string) Option {
	return func(ci *fs.ConfigInfo) {
		ci.ClientCert = certFile
		ci.ClientKey = keyFile
		ci.ClientPass = password
	}
}
