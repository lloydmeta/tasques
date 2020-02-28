package tracing

import (
	"context"

	"go.elastic.co/apm"

	"github.com/lloydmeta/tasques/internal/domain/tracing"
)

// Returns a thin wrapper around APM's tracing implementation
func NewTracer() tracing.Tracer {
	return &tracerImpl{getApmTracer: func() *apm.Tracer {
		return apm.DefaultTracer
	}}
}

type transactionsImpl struct {
	apmTx *apm.Transaction
}

func (t *transactionsImpl) Context() context.Context {
	return apm.ContextWithTransaction(context.Background(), t.apmTx)
}

func (t *transactionsImpl) End() {
	t.apmTx.End()
}

type tracerImpl struct {
	getApmTracer func() *apm.Tracer
}

func (t *tracerImpl) BackgroundTx(name string) tracing.Transaction {
	tracer := t.getApmTracer()
	tx := tracer.StartTransaction(name, "backgroundjob")
	return &transactionsImpl{apmTx: tx}
}

// <--- For testing

type noopTx struct{}

func (n noopTx) Context() context.Context {
	return context.Background()
}

func (n noopTx) End() {
}

type NoopTracer struct{}

func (n NoopTracer) BackgroundTx(name string) tracing.Transaction {
	return noopTx{}
}

// For testing -->
