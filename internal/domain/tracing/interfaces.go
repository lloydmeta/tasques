package tracing

import "context"

type Transaction interface {
	Context() context.Context
	End()
}

type Tracer interface {
	BackgroundTx(name string) Transaction
}
