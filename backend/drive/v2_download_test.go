package drive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
	"zclone/fs"
	"zclone/lib/pacer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	drivev2 "google.golang.org/api/drive/v2"
	drivev3 "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type downloadTransport func(*http.Request) (*http.Response, error)

func (f downloadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type downloadBody struct {
	io.Reader
	closed bool
}

func (b *downloadBody) Close() error { b.closed = true; return nil }

func TestV2DownloadURL(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"success", 200, `{"downloadUrl":"https://download.invalid/file","ignored":true}`},
		{"missing URL", 200, `{}`},
		{"malformed JSON", 200, `{`},
		{"rate limit", 403, `{"error":{"code":403,"message":"slow down","errors"
 "fmt":[{"reason":"rateLimitExceeded"}]}}`},
		{"server error", 503, `unavailable`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests []*http.Request
			var bodies []*downloadBody
			client := &http.Client{Transport: downloadTransport(func(r *http.Request) (*http.Response, error) {
				requests = append(requests, r)
				b := &downloadBody{Reader: strings.NewReader(tc.body)}
				bodies = append(bodies, b)
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Retry-After": []string{"1"}}, Body: b}, nil
			})}
			service, err := drivev2.NewService(context.Background(), option.WithHTTPClient(client))
			require.NoError(t, err)
			reference, referenceErr := service.Files.Get("id /?#%").Fields("downloadUrl").SupportsAllDrives(true).Context(context.Background()).Do()
			f := &Fs{svc: &drivev3.Service{BasePath: "https://www.googleapis.com/drive/v3/"}, client: client}
			got, err := f.getV2DownloadURL(context.Background(), "id /?#%")
			require.Len(t, requests, 2)
			require.Equal(t, requests[0].Method, requests[1].Method)
			require.Equal(t, requests[0].URL.String(), requests[1].URL.String())
			require.Equal(t, requests[0].Header.Get("User-Agent"), requests[1].Header.Get("User-Agent"))
			require.Equal(t, referenceErr == nil, err == nil)
			if referenceErr == nil {
				require.Equal(t, reference.DownloadUrl, got)
			}
			var expectedGoogleError, gotGoogleError *googleapi.Error
			if errors.As(referenceErr, &expectedGoogleError) {
				require.ErrorAs(t, err, &gotGoogleError)
				require.Equal(t, expectedGoogleError.Code, gotGoogleError.Code)
				require.Equal(t, expectedGoogleError.Errors, gotGoogleError.Errors)
				require.Equal(t, expectedGoogleError.Header, gotGoogleError.Header)
			}
			for _, b := range bodies {
				require.True(t, b.closed, "response body must close")
			}
		})
	}
}

func TestV2DownloadURLCancellationAndClientChange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &Fs{svc: &drivev3.Service{BasePath: "https://www.googleapis.com/drive/v3/"}, client: &http.Client{Transport: downloadTransport(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})}}
	_, err := f.getV2DownloadURL(ctx, "id")
	require.ErrorIs(t, err, context.Canceled)
	f.client = &http.Client{Transport: downloadTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"downloadUrl":"https://changed.invalid/file"}`))}, nil
	})}
	got, err := f.getV2DownloadURL(context.Background(), "id")
	require.NoError(t, err)
	require.Equal(t, "https://changed.invalid/file", got)
}

func TestV2DownloadOpen(t *testing.T) {
	for _, failMetadata := range []bool{false, true} {
		t.Run(fmt.Sprint(failMetadata), func(t *testing.T) {
			metadataCalls := 0
			client := &http.Client{Transport: downloadTransport(func(r *http.Request) (*http.Response, error) {
				status, body := 200, "payload"
				if r.URL.Host == "www.googleapis.com" {
					metadataCalls++
					if failMetadata {
						status, body = 404, `{"error":{"code":404,"message":"missing"}}`
					} else if metadataCalls == 1 {
						status, body = 503, `{"error":{"code":503,"message":"retry"}}`
					} else {
						body = `{"downloadUrl":"https://new.invalid/file"}`
					}
				} else {
					assert.Equal(t, "bytes=0-6", r.Header.Get("Range"))
					if failMetadata {
						assert.Equal(t, "original.invalid", r.URL.Host)
					} else {
						assert.Equal(t, "new.invalid", r.URL.Host)
					}
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			f := &Fs{svc: &drivev3.Service{BasePath: "https://www.googleapis.com/drive/v3/"}, client: client, pacer: fs.NewPacer(context.Background(), pacer.NewGoogleDrive(pacer.MinSleep(time.Millisecond), pacer.Burst(100)))}
			o := &Object{baseObject: baseObject{fs: f, id: "id", bytes: 7}, url: "https://original.invalid/file", v2Download: true}
			in, err := o.Open(context.Background(), &fs.RangeOption{Start: 0, End: 6})
			require.NoError(t, err)
			data, err := io.ReadAll(in)
			require.NoError(t, err)
			require.NoError(t, in.Close())
			require.Equal(t, "payload", string(data))
			if failMetadata {
				require.Equal(t, 1, metadataCalls)
				require.True(t, o.v2Download)
			} else {
				require.Equal(t, 2, metadataCalls)
				require.False(t, o.v2Download)
			}
		})
	}
}

func TestV2DownloadEndpoint(t *testing.T) {
	for _, root := range []string{"https://www.mtls.googleapis.com/drive/", "https://www.example.invalid/drive/"} {
		f := &Fs{svc: &drivev3.Service{BasePath: root + "v3/"}, client: &http.Client{Transport: downloadTransport(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, root+"v2/files/id", strings.Split(r.URL.String(), "?")[0])
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"downloadUrl":"https://download.invalid"}`))}, nil
		})}}
		_, err := f.getV2DownloadURL(context.Background(), "id")
		require.NoError(t, err)
	}
}
