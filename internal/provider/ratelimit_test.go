package provider

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caveman/go-rt/internal/domain"
	"golang.org/x/sync/errgroup"
)

type mockProvider struct {
	id          string
	activeCalls int64
	maxObserved int64
	delay       time.Duration
}

func (m *mockProvider) ID() string {
	return m.id
}

func (m *mockProvider) Call(ctx context.Context, req domain.PromptRequest) (domain.ProviderResponse, error) {
	curr := atomic.AddInt64(&m.activeCalls, 1)
	defer atomic.AddInt64(&m.activeCalls, -1)

	for {
		old := atomic.LoadInt64(&m.maxObserved)
		if curr <= old || atomic.CompareAndSwapInt64(&m.maxObserved, old, curr) {
			break
		}
	}
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return domain.ProviderResponse{Output: "mock-echo: " + req.Prompt}, nil
}

func TestRateLimitedProviderConcurrencyThrottle(t *testing.T) {
	mock := &mockProvider{id: "mock:test", delay: 20 * time.Millisecond}
	maxConcurrency := 2
	guarded := NewRateLimitedProvider(mock, 0, 0, maxConcurrency)

	var g errgroup.Group
	for i := 0; i < 6; i++ {
		g.Go(func() error {
			_, err := guarded.Call(context.Background(), domain.PromptRequest{Prompt: "ping"})
			return err
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatalf("unexpected group error: %v", err)
	}
	observed := atomic.LoadInt64(&mock.maxObserved)
	if observed > int64(maxConcurrency) {
		t.Errorf("max concurrent calls %d exceeded limit %d", observed, maxConcurrency)
	}
}

func TestRateLimitedProviderContextCancel(t *testing.T) {
	mock := &mockProvider{id: "mock:test", delay: 100 * time.Millisecond}
	guarded := NewRateLimitedProvider(mock, 0, 0, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Fill the single slot
	go func() {
		_, _ = guarded.Call(context.Background(), domain.PromptRequest{Prompt: "occupy"})
	}()
	time.Sleep(5 * time.Millisecond)

	// Second call should timeout
	_, err := guarded.Call(ctx, domain.PromptRequest{Prompt: "waiter"})
	if err == nil {
		t.Fatal("expected context timeout error, got nil")
	}
}
