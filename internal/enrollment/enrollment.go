package enrollment

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	ironcrypto "github.com/parisnakitakejser/ironroot/internal/crypto"
	"github.com/parisnakitakejser/ironroot/internal/db"
	"github.com/parisnakitakejser/ironroot/internal/telemetry"
)

var ErrInvalidToken = errors.New("invalid bootstrap token")

type Service struct{ Store db.Store }

type Request struct {
	Token     string `json:"token"`
	Hostname  string `json:"hostname"`
	MachineID string `json:"machine_id"`
	MAC       string `json:"mac,omitempty"`
}

type Response struct {
	EnrollmentID string `json:"enrollment_id"`
}

func (s Service) ValidateAndEnroll(ctx context.Context, req Request) (Response, error) {
	ctx, span := telemetry.StartSpan(ctx, "enrollment.validate")
	if strings.TrimSpace(req.Hostname) == "" || strings.TrimSpace(req.MachineID) == "" {
		err := errors.New("hostname and machine_id are required")
		telemetry.EndSpan(span, err)
		return Response{}, err
	}
	tokenCtx, tokenSpan := telemetry.StartSpan(ctx, "enrollment.validate_bootstrap_token")
	token, err := s.Store.GetBootstrapTokenByHash(ctx, ironcrypto.HashToken(req.Token))
	if err != nil || token.RevokedAt != nil || time.Now().After(token.ExpiresAt) || !strings.EqualFold(token.Hostname, req.Hostname) {
		telemetry.EndSpan(tokenSpan, ErrInvalidToken)
		telemetry.EndSpan(span, ErrInvalidToken)
		return Response{}, ErrInvalidToken
	}
	telemetry.EndSpan(tokenSpan, nil)
	id := uuid.NewString()
	err = s.Store.CreateEnrollment(tokenCtx, db.Enrollment{
		ID: id, Hostname: req.Hostname, MachineID: req.MachineID, MAC: req.MAC, TokenID: token.ID, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		telemetry.EndSpan(span, err)
		return Response{}, err
	}
	telemetry.EndSpan(span, nil)
	return Response{EnrollmentID: id}, nil
}
