package native

import (
	"context"
	"errors"
	"fmt"
	"gravity/internal/client"
	"io"
	"net/http"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/fserrors"
	"github.com/rclone/rclone/fs/hash"
	"github.com/rclone/rclone/lib/pacer"
	"github.com/rclone/rclone/lib/rest"
)

var (
	retryErrorCodes = []int{
		429,
		500, 502, 503, 504,
	}
	errorReadOnly = errors.New("object is read only")
)

func shouldRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if fserrors.ContextError(ctx, &err) {
		return false, err
	}
	return fserrors.ShouldRetry(err) || fserrors.ShouldRetryHTTP(resp, retryErrorCodes), err
}

type Option func(*HTTPObject)

func WithURL(url string) Option {
	return func(o *HTTPObject) {
		o.url = url
	}
}
func WithRemote(remote string) Option {
	return func(o *HTTPObject) {
		o.remote = remote
	}
}
func WithSize(size int64) Option {
	return func(o *HTTPObject) {
		o.size = size
	}
}
func WithModTime(modTime time.Time) Option {
	return func(o *HTTPObject) {
		o.modTime = modTime
	}
}
func WithRetries(retries int) Option {
	return func(o *HTTPObject) {
		o.retries = retries
	}
}
func WithClient(client *client.Client) Option {
	return func(o *HTTPObject) {
		o.client = client
	}
}

type HTTPObject struct {
	p        *fs.Pacer
	client   *client.Client
	url      string
	remote   string
	size     int64
	modTime  time.Time
	mimeType string
	retries  int
}

func NewHTTPObject(ctx context.Context, opts ...Option) *HTTPObject {
	o := &HTTPObject{}
	for _, opt := range opts {
		opt(o)
	}
	o.p = fs.NewPacer(ctx, pacer.NewDefault())
	o.p.SetRetries(o.retries)
	return o
}

func (o *HTTPObject) String() string {
	if o == nil {
		return "<nil>"
	}
	return o.remote
}

func (o *HTTPObject) Remote() string {
	return o.remote
}

func (o *HTTPObject) ModTime(ctx context.Context) time.Time {
	return o.modTime
}

func (o *HTTPObject) Size() int64 {
	return o.size
}

func (o *HTTPObject) Fs() fs.Info {
	return nil
}

func (o *HTTPObject) Storable() bool {
	return true
}

func (o *HTTPObject) Open(ctx context.Context, options ...fs.OpenOption) (io.ReadCloser, error) {
	var (
		err error
		res *http.Response
	)
	fs.FixRangeOption(options, o.size)
	err = o.p.Call(func() (bool, error) {
		opts := rest.Opts{
			Method:  "GET",
			RootURL: o.url,
			Options: options,
		}
		res, err = o.client.Call(ctx, &opts)
		return shouldRetry(ctx, res, err)
	})

	if err != nil {
		return nil, fmt.Errorf("Open failed: %w", err)
	}
	return res.Body, nil
}

func (o *HTTPObject) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	return errorReadOnly
}

func (o *HTTPObject) Remove(ctx context.Context) error {
	return errorReadOnly
}

func (o *HTTPObject) SetModTime(ctx context.Context, modTime time.Time) error {
	return errorReadOnly
}

func (o *HTTPObject) Hash(ctx context.Context, r hash.Type) (string, error) {
	return "", hash.ErrUnsupported
}

var (
	_ fs.Object = &HTTPObject{}
)
