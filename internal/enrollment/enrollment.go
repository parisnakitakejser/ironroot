package enrollment

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	ironcrypto "github.com/ironroot/ironroot/internal/crypto"
	"github.com/ironroot/ironroot/internal/db"
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
	if strings.TrimSpace(req.Hostname) == "" || strings.TrimSpace(req.MachineID) == "" {
		return Response{}, errors.New("hostname and machine_id are required")
	}
	token, err := s.Store.GetBootstrapTokenByHash(ctx, ironcrypto.HashToken(req.Token))
	if err != nil || token.RevokedAt != nil || time.Now().After(token.ExpiresAt) || !strings.EqualFold(token.Hostname, req.Hostname) {
		return Response{}, ErrInvalidToken
	}
	id := uuid.NewString()
	err = s.Store.CreateEnrollment(ctx, db.Enrollment{
		ID: id, Hostname: req.Hostname, MachineID: req.MachineID, MAC: req.MAC, TokenID: token.ID, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return Response{}, err
	}
	return Response{EnrollmentID: id}, nil
}
