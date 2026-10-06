package dl

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShrutiRetryDelayHonorsRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{"3"}},
	}

	if got := shrutiRetryDelay(resp, 1); got != 3*time.Second {
		t.Fatalf("expected 3s, got %v", got)
	}
}

func TestShrutiRetryDelayBacksOffWithoutRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     make(http.Header),
	}

	if got := shrutiRetryDelay(resp, 2); got != 4*time.Second {
		t.Fatalf("expected 4s, got %v", got)
	}
}

func TestShrutiHTTPClientUsesHTTP11(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 1 || r.ProtoMinor != 1 {
			t.Fatalf("expected HTTP/1.1, got %s", r.Proto)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newShrutiHTTPClient(time.Second)
	client.Transport.(*http.Transport).TLSClientConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestShrutiDownloadLockSerializesRequests(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	done := make(chan struct{})

	go func() {
		shrutiDownloadMu.Lock()
		started <- struct{}{}
		<-release
		shrutiDownloadMu.Unlock()
		close(done)
	}()

	<-started

	acquired := make(chan struct{})
	go func() {
		shrutiDownloadMu.Lock()
		close(acquired)
		shrutiDownloadMu.Unlock()
	}()

	select {
	case <-acquired:
		t.Fatal("second Shruti download acquired lock while first was active")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	<-done

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second Shruti download did not acquire lock after first finished")
	}
}
