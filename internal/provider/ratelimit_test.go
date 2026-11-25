package provider

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/bkelceoglu/go-rt/internal/domain"
	"golang.org/x/sync/errgroup"
)

var _ domain.Provider = (*mockProvider)(nil)

type mockProvider struct {
	id          string
	activeCalls atomic.Int64
	maxObserved atomic.Int64
	delay       time.Duration
}

func (m *mockProvider) ID() string {
	return m.id
}

func (m *mockProvider) Call(_ context.Context, req domain.PromptRequest) (domain.ProviderResponse, error) {
	curr := m.activeCalls.Add(1)
	defer m.activeCalls.Add(-1)

	for {
		old := m.maxObserved.Load()
		if curr <= old || m.maxObserved.CompareAndSwap(old, curr) {
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
	for range 6 {
		g.Go(func() error {
			_, err := guarded.Call(t.Context(), domain.PromptRequest{Prompt: "ping"})
			return err
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatalf("unexpected group error: %v", err)
	}
	observed := mock.maxObserved.Load()
	if observed > int64(maxConcurrency) {
		t.Errorf("max concurrent calls %d exceeded limit %d", observed, maxConcurrency)
	}
}

func TestRateLimitedProviderContextCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mock := &mockProvider{id: "mock:test", delay: 100 * time.Millisecond}
		guarded := NewRateLimitedProvider(mock, 0, 0, 1)

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Go(func() {
			if _, err := guarded.Call(t.Context(), domain.PromptRequest{Prompt: "occupy"}); err != nil {
				t.Errorf("occupy call failed: %v", err)
			}
		})
		time.Sleep(5 * time.Millisecond)
		synctest.Wait()

		_, err := guarded.Call(ctx, domain.PromptRequest{Prompt: "waiter"})
		if err == nil {
			t.Fatal("expected context timeout error, got nil")
		}
		wg.Wait()
	})
}
