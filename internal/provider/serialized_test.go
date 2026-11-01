package provider

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caveman/go-rt/internal/domain"
)

type concurrencyProbeProvider struct {
	id         string
	active     atomic.Int32
	maxOverlap atomic.Int32
}

func (c *concurrencyProbeProvider) ID() string { return c.id }

func (c *concurrencyProbeProvider) Call(_ context.Context, _ domain.PromptRequest) (domain.ProviderResponse, error) {
	cur := c.active.Add(1)
	for {
		prev := c.maxOverlap.Load()
		if cur <= prev || c.maxOverlap.CompareAndSwap(prev, cur) {
			break
		}
	}
	time.Sleep(15 * time.Millisecond)
	c.active.Add(-1)
	return domain.ProviderResponse{Output: "ok"}, nil
}

func TestWrapSharedSequentialPreventsConcurrentOverlap(t *testing.T) {
	probe := &concurrencyProbeProvider{id: "llamacpp:chat:Qwen2.5-7B"}
	attacker, judge := WrapSharedSequential(probe, probe, true)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = attacker.Call(context.Background(), domain.PromptRequest{Prompt: "attack"})
		}()
		go func() {
			defer wg.Done()
			_, _ = judge.Call(context.Background(), domain.PromptRequest{Prompt: "judge"})
		}()
	}
	wg.Wait()

	if probe.maxOverlap.Load() > 1 {
		t.Fatalf("expected maxOverlap=1 under shared sequential lock, got %d", probe.maxOverlap.Load())
	}
}
