package nrredis

import (
	"context"
	"strings"

	"github.com/go-redis/redis/v8"
	newrelic "github.com/newrelic/go-agent"
)

const startTimeKey = iota

// WrapRedisClient adds newrelic measurements for commands and returns cloned client
func WrapRedisClient(txn newrelic.Transaction, c *redis.Client) *redis.Client {
	if txn == nil {
		return c
	}

	// clone using context
	ctx := c.Context()
	copy := c.WithContext(ctx)
	copy.AddHook(hook{txn: txn})

	return copy
}

type segment interface {
	End() error
}

// create segment through function to be able to test it
var segmentBuilder = func(txn newrelic.Transaction, product newrelic.DatastoreProduct, operation string) segment {
	return &newrelic.DatastoreSegment{
		StartTime: newrelic.StartSegmentNow(txn),
		Product:   product,
		Operation: operation,
	}
}

type hook struct {
	txn newrelic.Transaction
}

func (h hook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	return context.WithValue(ctx, startTimeKey, newrelic.StartSegmentNow(h.txn)), nil
}

func (h hook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	return segmentBuilder(h.txn, newrelic.DatastoreRedis, strings.Split(cmd.String(), " ")[0]).End()
}

func (h hook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (h hook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	return nil
}
