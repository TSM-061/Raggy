package kafkawork

import (
	"context"
	"errors"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"testing"
)

type fakeCommitter struct {
	commits  int
	firstErr bool
}

func (f *fakeCommitter) CommitRecords(context.Context, ...*kgo.Record) error {
	f.commits++
	if f.firstErr && f.commits == 1 {
		return errors.New("offline")
	}
	return nil
}
func TestRetryProcessingBeforeCommit(t *testing.T) {
	f := &fakeCommitter{}
	calls := 0
	if !ProcessRecord(context.Background(), f, &kgo.Record{}, func(context.Context, *kgo.Record) error {
		calls++
		if calls == 1 {
			return errors.New("temporary")
		}
		return nil
	}) {
		t.Fatal("failed")
	}
	if calls != 2 || f.commits != 1 {
		t.Fatalf("calls=%d commits=%d", calls, f.commits)
	}
}
func TestCommitRetryDoesNotRepeatProcessing(t *testing.T) {
	f := &fakeCommitter{firstErr: true}
	calls := 0
	ProcessRecord(context.Background(), f, &kgo.Record{}, func(context.Context, *kgo.Record) error { calls++; return nil })
	if calls != 1 || f.commits != 2 {
		t.Fatalf("calls=%d commits=%d", calls, f.commits)
	}
}
func TestFailedProcessingNeverCommitsOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &fakeCommitter{}
	if ProcessRecord(ctx, f, &kgo.Record{}, func(context.Context, *kgo.Record) error { cancel(); return errors.New("failed") }) || f.commits != 0 {
		t.Fatal("failed work committed")
	}
}
func TestInvalidRecordIsRejectedWithoutRetry(t *testing.T) {
	f := &fakeCommitter{}
	if !ProcessRecord(context.Background(), f, &kgo.Record{}, func(context.Context, *kgo.Record) error { return serviceerr.InvalidInput }) || f.commits != 1 {
		t.Fatal("invalid record not acknowledged")
	}
}
