package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/ironroot/ironroot/internal/db"
)

type Logger struct{ Store db.Store }

func New(store db.Store) Logger { return Logger{Store: store} }

func (l Logger) Record(ctx context.Context, action, actor, target string, metadata any) {
	var meta string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		meta = string(b)
	}
	span := trace.SpanContextFromContext(ctx)
	_ = l.Store.CreateAuditLog(ctx, db.AuditLog{
		ID: uuid.NewString(), Action: action, Actor: actor, Target: target, Metadata: meta, TraceID: span.TraceID().String(), CreatedAt: time.Now().UTC(),
	})
}
