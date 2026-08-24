package app

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aivo/core/domain"
)

func TestExecuteBoundedParallelToolCallsHonorsLimitAndOrder(t *testing.T) {
	calls := []domain.ChatToolCall{{ID: "one", Name: "agent_delegate_task"}, {ID: "two", Name: "agent_delegate_task"}, {ID: "three", Name: "agent_delegate_task"}}
	var active atomic.Int32
	var maximum atomic.Int32
	results := executeBoundedParallelToolCalls(context.Background(), calls, 2, func(call domain.ChatToolCall) domain.ToolResult {
		current := active.Add(1)
		for {
			seen := maximum.Load()
			if current <= seen || maximum.CompareAndSwap(seen, current) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
		active.Add(-1)
		return domain.ToolResult{CallID: call.ID, Name: call.Name, OK: true, Content: call.ID}
	})
	if maximum.Load() != 2 {
		t.Fatalf("maximum concurrency = %d, want 2", maximum.Load())
	}
	for index := range calls {
		if results[index].CallID != calls[index].ID {
			t.Fatalf("results lost request order: %#v", results)
		}
	}
}

func TestExecuteBoundedParallelToolCallsStopsQueuedWorkAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := []domain.ChatToolCall{{ID: "one", Name: "task"}, {ID: "two", Name: "task"}, {ID: "three", Name: "task"}}
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var executions atomic.Int32
	var once sync.Once
	go func() {
		<-started
		cancel()
		close(release)
	}()
	results := executeBoundedParallelToolCalls(ctx, calls, 1, func(call domain.ChatToolCall) domain.ToolResult {
		executions.Add(1)
		once.Do(func() { started <- struct{}{} })
		<-release
		return domain.ToolResult{CallID: call.ID, Name: call.Name, OK: true}
	})
	if executions.Load() != 1 {
		t.Fatalf("executions = %d, want only active child", executions.Load())
	}
	cancelled := 0
	for _, result := range results {
		if !result.OK {
			cancelled++
		}
	}
	if cancelled != 2 {
		t.Fatalf("results = %#v, want two queued cancellations", results)
	}
}
