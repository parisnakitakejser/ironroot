package ca

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ironroot/ironroot/internal/config"
)

type Authority interface {
	SignCSR(csrPEM string, dnsNames []string, lifetime time.Duration) (Issued, error)
	ChainPEM() string
	RootPEM() string
	CAID() string
}

type Issued struct {
	Serial    string
	CertPEM   string
	NotBefore time.Time
	NotAfter  time.Time
}

type DisabledAuthority struct{}

func (DisabledAuthority) SignCSR(string, []string, time.Duration) (Issued, error) {
	return Issued{}, errors.New("CA authority is not configured")
}
func (DisabledAuthority) ChainPEM() string { return "" }
func (DisabledAuthority) RootPEM() string  { return "" }
func (DisabledAuthority) CAID() string     { return "unconfigured" }

type FileAuthority struct {
	cert  *x509.Certificate
	key   crypto.Signer
	chain string
	root  string
	caID  string
}

func LoadAuthority(cfg config.PKIConfig) (*FileAuthority, error) {
	certPEM, err := os.ReadFile(cfg.IntermediateCertFile)
	if err != nil {
		return nil, err
	}
	keyPEM, err := os.ReadFile(cfg.IntermediateKeyFile)
	if err != nil {
		return nil, err
	}
	certBlock, _ := pem.Decode(certPEM)
	keyBlock, _ := pem.Decode(keyPEM)
	if certBlock == nil || keyBlock == nil {
		return nil, errors.New("invalid CA PEM material")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, err
	}
	var key any
	if x509.IsEncryptedPEMBlock(keyBlock) {
		if cfg.IntermediateKeyPass == "" {
			return nil, errors.New("encrypted intermediate key requires pki.intermediate_key_pass")
		}
		der, err := x509.DecryptPEMBlock(keyBlock, []byte(cfg.IntermediateKeyPass))
		if err != nil {
			return nil, err
		}
		key, err = x509.ParsePKCS8PrivateKey(der)
		if err != nil {
			return nil, err
		}
	} else {
		key, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		}
		if err != nil {
			return nil, err
		}
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, errors.New("intermediate key is not a signer")
	}
	chainBytes, _ := os.ReadFile(cfg.ChainFile)
	rootBytes, _ := os.ReadFile(cfg.RootFile)
	fp := sha256.Sum256(cert.Raw)
	return &FileAuthority{cert: cert, key: signer, chain: string(chainBytes), root: string(rootBytes), caID: hex.EncodeToString(fp[:])}, nil
}

func (a *FileAuthority) SignCSR(csrPEM string, dnsNames []string, lifetime time.Duration) (Issued, error) {
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil {
		return Issued{}, errors.New("invalid CSR PEM")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return Issued{}, err
	}
	if err := csr.CheckSignature(); err != nil {
		return Issued{}, err
	}
	if len(dnsNames) == 0 {
		dnsNames = csr.DNSNames
	}
	for _, name := range dnsNames {
		if strings.TrimSpace(name) == "" {
			return Issued{}, errors.New("dns names must not be empty")
		}
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return Issued{}, err
	}
	notBefore := time.Now().UTC().Add(-1 * time.Minute)
	notAfter := notBefore.Add(lifetime)
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      csr.Subject,
		DNSNames:     dnsNames,
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, a.cert, csr.PublicKey, a.key)
	if err != nil {
		return Issued{}, err
	}
	return Issued{
		Serial: serial.Text(16), CertPEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), NotBefore: notBefore, NotAfter: notAfter,
	}, nil
}

func (a *FileAuthority) ChainPEM() string { return a.chain }
func (a *FileAuthority) RootPEM() string  { return a.root }
func (a *FileAuthority) CAID() string     { return a.caID }

func GenerateKeyAndCSR(commonName string, dnsNames []string) ([]byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	tpl := &x509.CertificateRequest{DNSNames: dnsNames}
	tpl.Subject.CommonName = commonName
	csr, err := x509.CreateCertificateRequest(rand.Reader, tpl, key)
	if err != nil {
		return nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
	return keyPEM, csrPEM, nil
}
