package ca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	rootKeyName          = "root-ca.key"
	rootCertName         = "root-ca.crt"
	intermediateKeyName  = "intermediate-ca.key"
	intermediateCertName = "intermediate-ca.crt"
	intermediateCSRName  = "intermediate-ca.csr"
	chainName            = "ca-chain.crt"
)

type CreateRootOptions struct {
	Name     string
	OutDir   string
	Password string
	Lifetime time.Duration
	Now      time.Time
}

type CreateIntermediateOptions struct {
	Name         string
	OutDir       string
	RootCertPath string
	RootKeyPath  string
	RootPassword string
	Password     string
	Lifetime     time.Duration
	Now          time.Time
}

type CreateRootResult struct {
	KeyPath      string
	CertPath     string
	Fingerprint  string
	PasswordFile string
	NotAfter     time.Time
}

type CreateIntermediateResult struct {
	KeyPath      string
	CSRPath      string
	CertPath     string
	ChainPath    string
	Fingerprint  string
	PasswordFile string
	NotAfter     time.Time
}

type CertInfo struct {
	Path        string    `json:"path"`
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	IsCA        bool      `json:"is_ca"`
	Fingerprint string    `json:"fingerprint"`
}

func CreateRoot(opts CreateRootOptions) (CreateRootResult, error) {
	if opts.Name == "" {
		opts.Name = "IronRoot Local Root CA"
	}
	if opts.OutDir == "" {
		opts.OutDir = "."
	}
	if opts.Lifetime == 0 {
		opts.Lifetime = 20 * 365 * 24 * time.Hour
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.Password == "" {
		return CreateRootResult{}, errors.New("root key password is required")
	}
	if err := os.MkdirAll(opts.OutDir, 0o700); err != nil {
		return CreateRootResult{}, err
	}
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return CreateRootResult{}, err
	}
	serial, err := randomSerial()
	if err != nil {
		return CreateRootResult{}, err
	}
	tpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: opts.Name, Organization: []string{"IronRoot"}},
		NotBefore:             opts.Now.Add(-1 * time.Minute),
		NotAfter:              opts.Now.Add(opts.Lifetime),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return CreateRootResult{}, err
	}
	keyPath := filepath.Join(opts.OutDir, rootKeyName)
	certPath := filepath.Join(opts.OutDir, rootCertName)
	if err := writeEncryptedPKCS8Key(keyPath, key, opts.Password); err != nil {
		return CreateRootResult{}, err
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return CreateRootResult{}, err
	}
	return CreateRootResult{KeyPath: keyPath, CertPath: certPath, Fingerprint: fingerprint(der), NotAfter: tpl.NotAfter}, nil
}

func CreateIntermediate(opts CreateIntermediateOptions) (CreateIntermediateResult, error) {
	if opts.Name == "" {
		opts.Name = "IronRoot Local Intermediate CA"
	}
	if opts.OutDir == "" {
		opts.OutDir = "."
	}
	if opts.Lifetime == 0 {
		opts.Lifetime = 5 * 365 * 24 * time.Hour
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.RootPassword == "" {
		return CreateIntermediateResult{}, errors.New("root key password is required")
	}
	if opts.Password == "" {
		return CreateIntermediateResult{}, errors.New("intermediate key password is required")
	}
	if err := os.MkdirAll(opts.OutDir, 0o700); err != nil {
		return CreateIntermediateResult{}, err
	}
	rootCert, rootPEM, err := readCert(opts.RootCertPath)
	if err != nil {
		return CreateIntermediateResult{}, err
	}
	rootKey, err := readPrivateKey(opts.RootKeyPath, opts.RootPassword)
	if err != nil {
		return CreateIntermediateResult{}, err
	}
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return CreateIntermediateResult{}, err
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: opts.Name, Organization: []string{"IronRoot"}}}, key)
	if err != nil {
		return CreateIntermediateResult{}, err
	}
	serial, err := randomSerial()
	if err != nil {
		return CreateIntermediateResult{}, err
	}
	tpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: opts.Name, Organization: []string{"IronRoot"}},
		NotBefore:             opts.Now.Add(-1 * time.Minute),
		NotAfter:              opts.Now.Add(opts.Lifetime),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}
	if tpl.NotAfter.After(rootCert.NotAfter) {
		return CreateIntermediateResult{}, fmt.Errorf("intermediate lifetime exceeds root CA expiration %s", rootCert.NotAfter.Format(time.RFC3339))
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, rootCert, &key.PublicKey, rootKey)
	if err != nil {
		return CreateIntermediateResult{}, err
	}
	keyPath := filepath.Join(opts.OutDir, intermediateKeyName)
	csrPath := filepath.Join(opts.OutDir, intermediateCSRName)
	certPath := filepath.Join(opts.OutDir, intermediateCertName)
	chainPath := filepath.Join(opts.OutDir, chainName)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := writeEncryptedPKCS8Key(keyPath, key, opts.Password); err != nil {
		return CreateIntermediateResult{}, err
	}
	if err := os.WriteFile(csrPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}), 0o644); err != nil {
		return CreateIntermediateResult{}, err
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return CreateIntermediateResult{}, err
	}
	if err := os.WriteFile(chainPath, append(certPEM, rootPEM...), 0o644); err != nil {
		return CreateIntermediateResult{}, err
	}
	return CreateIntermediateResult{KeyPath: keyPath, CSRPath: csrPath, CertPath: certPath, ChainPath: chainPath, Fingerprint: fingerprint(der), NotAfter: tpl.NotAfter}, nil
}

func InspectCertificates(paths []string) ([]CertInfo, error) {
	var out []CertInfo
	for _, path := range paths {
		cert, _, err := readCert(path)
		if err != nil {
			return nil, err
		}
		out = append(out, CertInfo{
			Path: path, Subject: cert.Subject.String(), Issuer: cert.Issuer.String(),
			NotBefore: cert.NotBefore, NotAfter: cert.NotAfter, IsCA: cert.IsCA, Fingerprint: fingerprint(cert.Raw),
		})
	}
	return out, nil
}

func VerifyChain(rootPath, intermediatePath, leafPath string) error {
	root, _, err := readCert(rootPath)
	if err != nil {
		return err
	}
	intermediate, _, err := readCert(intermediatePath)
	if err != nil {
		return err
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	intermediates := x509.NewCertPool()
	if leafPath == "" {
		return intermediate.CheckSignatureFrom(root)
	}
	leaf, _, err := readCert(leafPath)
	if err != nil {
		return err
	}
	intermediates.AddCert(intermediate)
	opts := x509.VerifyOptions{Roots: roots, Intermediates: intermediates, CurrentTime: time.Now(), DNSName: firstDNSName(leaf)}
	_, err = leaf.Verify(opts)
	return err
}

func readCert(path string) (*x509.Certificate, []byte, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(body)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, nil, fmt.Errorf("%s does not contain a certificate PEM block", path)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	return cert, pem.EncodeToMemory(block), err
}

func readPrivateKey(path, password string) (*rsa.PrivateKey, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, fmt.Errorf("%s does not contain a private key PEM block", path)
	}
	der := block.Bytes
	if x509.IsEncryptedPEMBlock(block) {
		der, err = x509.DecryptPEMBlock(block, []byte(password))
		if err != nil {
			return nil, err
		}
	}
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return rsaKey, nil
}

func writeEncryptedPKCS8Key(path string, key *rsa.PrivateKey, password string) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	block, err := x509.EncryptPEMBlock(rand.Reader, "PRIVATE KEY", der, []byte(password), x509.PEMCipherAES256)
	if err != nil {
		return err
	}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0o600)
}

func randomSerial() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	encoded := strings.ToUpper(hex.EncodeToString(sum[:]))
	var parts []string
	for i := 0; i < len(encoded); i += 2 {
		parts = append(parts, encoded[i:i+2])
	}
	return strings.Join(parts, ":")
}

func firstDNSName(cert *x509.Certificate) string {
	if len(cert.DNSNames) > 0 {
		return cert.DNSNames[0]
	}
	for _, ip := range cert.IPAddresses {
		if ip != nil && !ip.Equal(net.IP{}) {
			return ""
		}
	}
	return ""
}
