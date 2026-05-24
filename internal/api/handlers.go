package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/parisnakitakejser/ironroot/internal/ca"
	"github.com/parisnakitakejser/ironroot/internal/db"
	"github.com/parisnakitakejser/ironroot/internal/enrollment"
	"github.com/parisnakitakejser/ironroot/internal/telemetry"
)

type handler struct{ dep Dependencies }

func (h handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (h handler) readyz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h handler) root(w http.ResponseWriter, r *http.Request) {
	writePEM(w, h.dep.Authority.RootPEM())
}

func (h handler) chain(w http.ResponseWriter, r *http.Request) {
	writePEM(w, h.dep.Authority.ChainPEM())
}

func (h handler) enroll(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "enrollment.workflow")
	defer span.End()
	var req enrollment.Request
	if !decode(w, r, &req) {
		return
	}
	resp, err := (enrollment.Service{Store: h.dep.Store}).ValidateAndEnroll(ctx, req)
	if err != nil {
		telemetry.Instruments().EnrollmentFailures.Add(r.Context(), 1)
		if errors.Is(err, enrollment.ErrInvalidToken) {
			telemetry.Instruments().TokenValidationFailures.Add(r.Context(), 1)
		}
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	telemetry.Instruments().Enrollments.Add(r.Context(), 1)
	h.dep.Audit.Record(ctx, "enrollment.created", req.Hostname, resp.EnrollmentID, map[string]string{"machine_id": req.MachineID})
	writeJSON(w, http.StatusCreated, resp)
}

type certificateRequest struct {
	EnrollmentID string   `json:"enrollment_id"`
	CSRPEM       string   `json:"csr_pem"`
	DNSNames     []string `json:"dns_names"`
}

type certificateResponse struct {
	Serial      string `json:"serial"`
	CertPEM     string `json:"cert_pem"`
	ChainPEM    string `json:"chain_pem"`
	RootPEM     string `json:"root_pem"`
	NotBefore   string `json:"not_before"`
	NotAfter    string `json:"not_after"`
	RenewBefore string `json:"renew_before"`
}

func (h handler) requestCertificate(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "certificate.issue.workflow")
	defer span.End()
	var req certificateRequest
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.EnrollmentID) == "" {
		writeError(w, http.StatusBadRequest, errors.New("enrollment_id is required"))
		return
	}
	en, err := h.dep.Store.GetEnrollment(ctx, req.EnrollmentID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, errors.New("unknown enrollment"))
		return
	}
	issued, err := h.dep.Authority.SignCSR(ctx, req.CSRPEM, req.DNSNames, h.dep.Config.PKI.DefaultLifetime)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.ensureCAConfig(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	err = h.dep.Store.StoreIssuedCertificate(ctx, db.IssuedCertificate{
		Serial: issued.Serial, CAID: h.dep.Authority.CAID(), EnrollmentID: en.ID, Subject: en.Hostname, DNSNames: strings.Join(req.DNSNames, ","), PEM: issued.CertPEM, NotBefore: issued.NotBefore, NotAfter: issued.NotAfter, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	attrs := metric.WithAttributes(attribute.String("certificate_type", "server"), attribute.String("issuer", telemetry.SanitizeLabel(h.dep.Authority.CAID())), attribute.String("status", "issued"))
	telemetry.Instruments().CertificatesIssued.Add(ctx, 1, attrs)
	telemetry.Instruments().ActiveCertificates.Add(ctx, 1, attrs)
	h.dep.Audit.Record(ctx, "certificate.issued", en.Hostname, issued.Serial, map[string]string{"enrollment_id": en.ID})
	writeJSON(w, http.StatusCreated, certResponse(h.dep, issued))
}

func (h handler) renewCertificate(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "certificate.renew.workflow")
	defer span.End()
	var req certificateRequest
	if !decode(w, r, &req) {
		return
	}
	oldSerial := r.URL.Query().Get("serial")
	if oldSerial != "" {
		old, err := h.dep.Store.GetIssuedCertificate(ctx, oldSerial)
		if err == nil && time.Until(old.NotAfter) > h.dep.Config.PKI.RenewBefore {
			writeError(w, http.StatusBadRequest, errors.New("certificate is not inside renewal window"))
			return
		}
	}
	issued, err := h.dep.Authority.SignCSR(ctx, req.CSRPEM, req.DNSNames, h.dep.Config.PKI.DefaultLifetime)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.ensureCAConfig(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := h.dep.Store.StoreIssuedCertificate(ctx, db.IssuedCertificate{Serial: issued.Serial, CAID: h.dep.Authority.CAID(), EnrollmentID: req.EnrollmentID, Subject: req.EnrollmentID, DNSNames: strings.Join(req.DNSNames, ","), PEM: issued.CertPEM, NotBefore: issued.NotBefore, NotAfter: issued.NotAfter, CreatedAt: time.Now().UTC()}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	attrs := metric.WithAttributes(attribute.String("certificate_type", "server"), attribute.String("issuer", telemetry.SanitizeLabel(h.dep.Authority.CAID())), attribute.String("status", "renewed"))
	telemetry.Instruments().CertificatesRenewed.Add(ctx, 1, attrs)
	telemetry.Instruments().ActiveCertificates.Add(ctx, 1, attrs)
	h.dep.Audit.Record(ctx, "certificate.renewed", req.EnrollmentID, issued.Serial, map[string]string{"previous_serial": oldSerial})
	writeJSON(w, http.StatusCreated, certResponse(h.dep, issued))
}

func (h handler) revokeCertificate(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.StartSpan(r.Context(), "certificate.revoke.workflow")
	defer span.End()
	var req struct {
		Serial string `json:"serial"`
		Reason string `json:"reason"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Serial == "" {
		writeError(w, http.StatusBadRequest, errors.New("serial is required"))
		return
	}
	if req.Reason == "" {
		req.Reason = "unspecified"
	}
	if err := h.dep.Store.RevokeCertificate(ctx, db.RevokedCertificate{Serial: req.Serial, Reason: req.Reason, RevokedAt: time.Now().UTC()}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	attrs := metric.WithAttributes(attribute.String("certificate_type", "server"), attribute.String("issuer", "unknown"), attribute.String("status", "revoked"))
	telemetry.Instruments().CertificatesRevoked.Add(ctx, 1, attrs)
	telemetry.Instruments().ActiveCertificates.Add(ctx, -1, attrs)
	h.dep.Audit.Record(ctx, "certificate.revoked", "admin", req.Serial, map[string]string{"reason": req.Reason})
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h handler) certificateStatus(w http.ResponseWriter, r *http.Request) {
	cert, err := h.dep.Store.GetIssuedCertificate(r.Context(), chi.URLParam(r, "serial"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, cert)
}

func (h handler) audit(w http.ResponseWriter, r *http.Request) {
	logs, err := h.dep.Store.ListAuditLogs(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func certResponse(dep Dependencies, issued ca.Issued) certificateResponse {
	return certificateResponse{Serial: issued.Serial, CertPEM: issued.CertPEM, ChainPEM: dep.Authority.ChainPEM(), RootPEM: dep.Authority.RootPEM(), NotBefore: issued.NotBefore.Format(time.RFC3339), NotAfter: issued.NotAfter.Format(time.RFC3339), RenewBefore: issued.NotAfter.Add(-dep.Config.PKI.RenewBefore).Format(time.RFC3339)}
}

func (h handler) ensureCAConfig(ctx context.Context) error {
	now := time.Now().UTC()
	return h.dep.Store.UpsertCAConfig(ctx, db.CAConfig{
		CAID:                    h.dep.Authority.CAID(),
		Name:                    "default",
		RootFingerprint:         h.dep.Authority.CAID(),
		IntermediateFingerprint: h.dep.Authority.CAID(),
		Status:                  "active",
		CreatedAt:               now,
		NotBefore:               now.Add(-time.Minute),
		NotAfter:                now.Add(5 * 365 * 24 * time.Hour),
	})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writePEM(w http.ResponseWriter, pem string) {
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(pem))
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error(), "request_id": uuid.NewString()})
}
