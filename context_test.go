package geoserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestContextCancellation verifies that an already-cancelled context causes
// resource methods to return without making a successful round-trip. The
// cancellation must propagate from the caller's context through DoRequestContext
// to the underlying http.Request and finally cause the transport to fail.
func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If we ever reach the server with a cancelled context, the test
		// is already broken — but slow the response anyway so an
		// immediate-cancel context wins the race.
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	gs := New(server.URL+"/", "u", "p", WithTimeout(5*time.Second))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call

	_, err := gs.GetWorkspacesContext(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	// The error should mention cancellation — Go's http transport surfaces
	// "context canceled" from the underlying RoundTripper.
	t.Logf("cancelled-context error: %v", err)
}

// TestContextDeadline verifies a sub-deadline shorter than the http.Client
// timeout still aborts the request.
func TestContextDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep so the deadline expires before we'd respond.
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	gs := New(server.URL+"/", "u", "p", WithTimeout(30*time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := gs.GetWorkspacesContext(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from expired deadline, got nil")
	}
	if elapsed > time.Second {
		t.Fatalf("request did not honor context deadline: took %v", elapsed)
	}
	t.Logf("deadline-expired error after %v: %v", elapsed, err)
}

// TestNewWithOptions verifies the New() constructor and option helpers.
func TestNewWithOptions(t *testing.T) {
	custom := &http.Client{Timeout: 7 * time.Second}
	gs := New("http://example/",
		"u",
		"p",
		WithHTTPClient(custom),
		WithUserAgent("test-agent/1.0"),
	)
	assert.Equal(t, "http://example/", gs.ServerURL)
	assert.Equal(t, "u", gs.Username)
	assert.Equal(t, "p", gs.Password)
	// WithHTTPClient sets a fresh client; WithUserAgent wraps its Transport.
	assert.NotNil(t, gs.HttpClient)
	assert.NotNil(t, gs.HttpClient.Transport)
	if _, ok := gs.HttpClient.Transport.(*userAgentTransport); !ok {
		t.Fatalf("expected userAgentTransport wrapping default transport, got %T", gs.HttpClient.Transport)
	}
}

// TestContextErrorIsRespected verifies that a non-2xx response from the
// httptest server is correctly translated to a typed *Error matchable via
// errors.Is.
func TestContextErrorIsRespected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"workspace not found"}`))
	}))
	t.Cleanup(server.Close)

	gs := New(server.URL+"/", "u", "p")

	_, err := gs.GetWorkspaceContext(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error from 404 response, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected errors.Is(err, ErrNotFound) == true, got false; err=%v", err)
	}

	// And error string preserves the historic "abstract:..." format.
	if !strings.Contains(err.Error(), "abstract:Not Found") {
		t.Fatalf("expected historic 'abstract:Not Found' format in error, got: %v", err)
	}
}
