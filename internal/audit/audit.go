package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/parisnakitakejser/ironroot/internal/db"
	"github.com/parisnakitakejser/ironroot/internal/telemetry"
)

type Logger struct{ Store db.Store }

func New(store db.Store) Logger { return Logger{Store: store} }

func (l Logger) Record(ctx context.Context, action, actor, target string, metadata any) {
	ctx, span := telemetry.StartSpan(ctx, "audit.write")
	var meta string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		meta = string(b)
	}
	sc := trace.SpanContextFromContext(ctx)
	err := l.Store.CreateAuditLog(ctx, db.AuditLog{
		ID: uuid.NewString(), Action: action, Actor: actor, Target: target, Metadata: meta, TraceID: sc.TraceID().String(), CreatedAt: time.Now().UTC(),
	})
	telemetry.EndSpan(span, err)
}
