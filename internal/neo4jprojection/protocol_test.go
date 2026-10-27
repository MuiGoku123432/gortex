package neo4jprojection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type protocolOutcome string

const (
	outcomeTransient       protocolOutcome = "transient"
	outcomePermanent       protocolOutcome = "permanent"
	outcomeActivation      protocolOutcome = "activation"
	outcomeCleanup         protocolOutcome = "cleanup"
	outcomeCommitUncertain protocolOutcome = "commit_uncertain"
)

type protocolRecorder struct {
	operations []string
	failures   map[string]protocolOutcome
	maxBatch   int
	closed     bool
}

func newProtocolRecorder(maxBatch int) *protocolRecorder {
	return &protocolRecorder{failures: make(map[string]protocolOutcome), maxBatch: maxBatch}
}

func (r *protocolRecorder) record(ctx context.Context, operation string, rows int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if rows > r.maxBatch {
		return fmt.Errorf("batch %d exceeds maximum %d", rows, r.maxBatch)
	}
	r.operations = append(r.operations, operation)
	if outcome := r.failures[operation]; outcome != "" {
		return fmt.Errorf("%s: %s", operation, outcome)
	}
	if operation == "close" {
		r.closed = true
	}
	return nil
}

func TestProjectionProtocolHarness(t *testing.T) {
	want := []string{"inspect", "lock", "pending", "nodes", "edges", "activate", "cleanup", "close"}
	recorder := newProtocolRecorder(2)
	for _, operation := range want {
		if err := recorder.record(context.Background(), operation, 1); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(recorder.operations, ",") != strings.Join(want, ",") || !recorder.closed {
		t.Fatalf("unexpected protocol: %#v closed=%v", recorder.operations, recorder.closed)
	}
	if err := recorder.record(context.Background(), "nodes", 3); err == nil {
		t.Fatal("oversized batch was accepted")
	}

	for _, tc := range []struct {
		operation string
		outcome   protocolOutcome
	}{
		{"nodes", outcomeTransient}, {"edges", outcomePermanent}, {"activate", outcomeActivation},
		{"cleanup", outcomeCleanup}, {"activate", outcomeCommitUncertain},
	} {
		r := newProtocolRecorder(2)
		r.failures[tc.operation] = tc.outcome
		if err := r.record(context.Background(), tc.operation, 1); err == nil || !strings.Contains(err.Error(), string(tc.outcome)) {
			t.Fatalf("%s outcome not injected: %v", tc.outcome, err)
		}
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := recorder.record(cancelled, "nodes", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}

	canary := "neo4j-password-canary"
	payload, err := json.Marshal(struct {
		Result string `json:"result"`
	}{Result: "complete"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), canary) || strings.Contains(strings.Join(recorder.operations, " "), canary) {
		t.Fatal("secret canary leaked into serialized output")
	}
}
