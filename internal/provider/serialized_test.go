package provider

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caveman/go-rt/internal/domain"
)

var _ domain.Provider = (*concurrencyProbeProvider)(nil)

type concurrencyProbeProvider struct {
	id         string
	active     atomic.Int32
	maxOverlap atomic.Int32
}

func (c *concurrencyProbeProvider) ID() string { return c.id }

func (c *concurrencyProbeProvider) Call(_ context.Context, _ domain.PromptRequest) (domain.ProviderResponse, error) {
	cur := c.active.Add(1)
	defer c.active.Add(-1)
	for {
		prev := c.maxOverlap.Load()
		if cur <= prev || c.maxOverlap.CompareAndSwap(prev, cur) {
			break
		}
	}
	time.Sleep(15 * time.Millisecond)
	return domain.ProviderResponse{Output: "ok"}, nil
}

func TestWrapSharedSequentialPreventsConcurrentOverlap(t *testing.T) {
	probe := &concurrencyProbeProvider{id: "llamacpp:chat:Qwen2.5-7B"}
	attacker, judge := WrapSharedSequential(probe, probe, true)

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			if _, err := attacker.Call(t.Context(), domain.PromptRequest{Prompt: "attack"}); err != nil {
				t.Errorf("attacker call failed: %v", err)
			}
		})
		wg.Go(func() {
			if _, err := judge.Call(t.Context(), domain.PromptRequest{Prompt: "judge"}); err != nil {
				t.Errorf("judge call failed: %v", err)
			}
		})
	}
	wg.Wait()

	if probe.maxOverlap.Load() > 1 {
		t.Fatalf("expected maxOverlap=1 under shared sequential lock, got %d", probe.maxOverlap.Load())
	}
}
