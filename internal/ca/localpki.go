package ca

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	icrypto "github.com/parisnakitakejser/ironroot/internal/crypto"
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
	Name                  string
	CommonName            string
	Organization          string
	OrganizationalUnit    string
	Country               string
	Province              string
	Locality              string
	Algorithm             string
	RSABits               int
	Curve                 string
	KeyFormat             string
	EncryptKey            bool
	Password              string
	Lifetime              time.Duration
	NotBefore             time.Time
	MaxPathLength         int
	IsCA                  bool
	AllowCertSigning      bool
	AllowCRLSigning       bool
	AllowServerAuth       bool
	AllowClientAuth       bool
	AllowCodeSigning      bool
	AllowEmailProtection  bool
	GenerateTrustBundle   bool
	WritePEM              bool
	WriteDER              bool
	Offline               bool
	BackupReminder        bool
	CRLEnabled            bool
	OCSPEnabled           bool
	PermittedDNSDomains   []string
	PermittedIPRanges     []string
	PermittedEmailDomains []string
	OutDir                string
	Now                   time.Time
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
	KeyPath          string
	CertPath         string
	PublicKeyPath    string
	PEMPath          string
	DERPath          string
	MetadataPath     string
	FingerprintsPath string
	RecoveryPath     string
	TrustBundleDir   string
	Fingerprint      string
	PasswordFile     string
	NotAfter         time.Time
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
	Path          string    `json:"path"`
	Subject       string    `json:"subject"`
	Issuer        string    `json:"issuer"`
	Algorithm     string    `json:"algorithm"`
	Curve         string    `json:"curve,omitempty"`
	KeySize       int       `json:"key_size,omitempty"`
	NotBefore     time.Time `json:"not_before"`
	NotAfter      time.Time `json:"not_after"`
	IsCA          bool      `json:"is_ca"`
	MaxPathLength int       `json:"max_path_length"`
	KeyUsage      []string  `json:"key_usage"`
	ExtKeyUsage   []string  `json:"extended_key_usage"`
	Fingerprint   string    `json:"fingerprint"`
	SKID          string    `json:"subject_key_id,omitempty"`
	AKID          string    `json:"authority_key_id,omitempty"`
}

func CreateRoot(opts CreateRootOptions) (CreateRootResult, error) {
	opts = defaultRootOptions(opts)
	if err := validateRootOptions(opts); err != nil {
		return CreateRootResult{}, err
	}
	if err := os.MkdirAll(opts.OutDir, 0o700); err != nil {
		return CreateRootResult{}, err
	}
	key, publicKey, err := generateSigner(opts.Algorithm, opts.RSABits, opts.Curve)
	if err != nil {
		return CreateRootResult{}, err
	}
	serial, err := randomSerial()
	if err != nil {
		return CreateRootResult{}, err
	}
	ipRanges, err := parseIPRanges(opts.PermittedIPRanges)
	if err != nil {
		return CreateRootResult{}, err
	}
	tpl := &x509.Certificate{
		SerialNumber:            serial,
		Subject:                 rootSubject(opts),
		NotBefore:               opts.NotBefore,
		NotAfter:                opts.Now.Add(opts.Lifetime),
		KeyUsage:                rootKeyUsage(opts),
		ExtKeyUsage:             rootExtKeyUsage(opts),
		BasicConstraintsValid:   true,
		IsCA:                    opts.IsCA,
		MaxPathLen:              opts.MaxPathLength,
		PermittedDNSDomains:     opts.PermittedDNSDomains,
		PermittedEmailAddresses: opts.PermittedEmailDomains,
		PermittedIPRanges:       ipRanges,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, publicKey, key)
	if err != nil {
		return CreateRootResult{}, err
	}
	keyPath := filepath.Join(opts.OutDir, rootKeyName)
	certPath := filepath.Join(opts.OutDir, rootCertName)
	if err := writePKCS8Key(keyPath, key, opts.Password, opts.EncryptKey); err != nil {
		return CreateRootResult{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return CreateRootResult{}, err
	}
	result := CreateRootResult{KeyPath: keyPath, CertPath: certPath, Fingerprint: fingerprint(der), NotAfter: tpl.NotAfter}
	if err := writeRootArtifacts(opts, &result, tpl, key, publicKey, der, certPEM); err != nil {
		return CreateRootResult{}, err
	}
	return result, nil
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
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
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
	if err := writePKCS8Key(keyPath, key, opts.Password, true); err != nil {
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
			Algorithm: keyAlgorithm(cert.PublicKey), Curve: keyCurve(cert.PublicKey), KeySize: keySize(cert.PublicKey),
			NotBefore: cert.NotBefore, NotAfter: cert.NotAfter, IsCA: cert.IsCA, MaxPathLength: cert.MaxPathLen,
			KeyUsage: keyUsageNames(cert.KeyUsage), ExtKeyUsage: extKeyUsageNames(cert.ExtKeyUsage),
			Fingerprint: fingerprint(cert.Raw), SKID: strings.ToUpper(hex.EncodeToString(cert.SubjectKeyId)),
			AKID: strings.ToUpper(hex.EncodeToString(cert.AuthorityKeyId)),
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

func readPrivateKey(path, password string) (crypto.Signer, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, fmt.Errorf("%s does not contain a private key PEM block", path)
	}
	der := block.Bytes
	if icrypto.IsEncryptedPEMBlockGCM(block) {
		der, err = icrypto.DecryptPrivateKeyGCM(block, password)
		if err != nil {
			return nil, err
		}
	}
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(der)
	}
	if err != nil {
		return nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, errors.New("private key is not a supported signer")
	}
	return signer, nil
}

func writePKCS8Key(path string, key crypto.Signer, password string, encrypt bool) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	var block *pem.Block
	if encrypt {
		if password == "" {
			return errors.New("private key password is required when encryption is enabled")
		}
		block, err = icrypto.EncryptPrivateKeyGCM(der, password)
		if err != nil {
			return err
		}
	} else {
		block = &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0o600)
}

func defaultRootOptions(opts CreateRootOptions) CreateRootOptions {
	if opts.Name == "" {
		opts.Name = "IronRoot Local Root CA"
	}
	if opts.CommonName == "" {
		opts.CommonName = opts.Name
	}
	if opts.Organization == "" {
		opts.Organization = "IronRoot"
	}
	if opts.Algorithm == "" {
		opts.Algorithm = "ecdsa"
	}
	opts.Algorithm = strings.ToLower(opts.Algorithm)
	if opts.RSABits == 0 {
		opts.RSABits = 4096
	}
	if opts.Curve == "" {
		opts.Curve = "p384"
	}
	opts.Curve = strings.ToLower(opts.Curve)
	if opts.KeyFormat == "" {
		opts.KeyFormat = "pkcs8"
	}
	opts.KeyFormat = strings.ToLower(opts.KeyFormat)
	if opts.Lifetime == 0 {
		opts.Lifetime = 20 * 365 * 24 * time.Hour
	}
	if opts.OutDir == "" {
		opts.OutDir = "."
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.NotBefore.IsZero() {
		opts.NotBefore = opts.Now.Add(-1 * time.Minute)
	}
	if opts.MaxPathLength == 0 {
		opts.MaxPathLength = 1
	}
	return opts
}

func validateRootOptions(opts CreateRootOptions) error {
	switch opts.Algorithm {
	case "ecdsa", "rsa":
	default:
		return fmt.Errorf("unsupported root key algorithm %q; use ecdsa or rsa", opts.Algorithm)
	}
	switch opts.KeyFormat {
	case "pkcs8", "pem":
	default:
		return fmt.Errorf("unsupported key format %q; use pkcs8", opts.KeyFormat)
	}
	if opts.Algorithm == "rsa" && opts.RSABits < 2048 {
		return errors.New("rsa root keys must be at least 2048 bits; 4096 is recommended")
	}
	if opts.Algorithm == "ecdsa" {
		if _, err := ellipticCurve(opts.Curve); err != nil {
			return err
		}
	}
	if opts.Lifetime <= 0 {
		return errors.New("root validity must be positive")
	}
	if opts.Lifetime > 30*365*24*time.Hour {
		return errors.New("root validity above 30 years is not supported by this workflow")
	}
	if !opts.IsCA {
		return errors.New("root certificate must be a CA certificate")
	}
	if !opts.AllowCertSigning {
		return errors.New("root certificate must allow certificate signing so it can sign Intermediate CAs")
	}
	if opts.MaxPathLength < 0 {
		return errors.New("max path length must be zero or greater")
	}
	return nil
}

func generateSigner(algorithm string, rsaBits int, curveName string) (crypto.Signer, any, error) {
	switch algorithm {
	case "rsa":
		key, err := rsa.GenerateKey(rand.Reader, rsaBits)
		if err != nil {
			return nil, nil, err
		}
		return key, &key.PublicKey, nil
	case "ecdsa":
		curve, err := ellipticCurve(curveName)
		if err != nil {
			return nil, nil, err
		}
		key, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return key, &key.PublicKey, nil
	default:
		return nil, nil, fmt.Errorf("unsupported key algorithm %q", algorithm)
	}
}

func ellipticCurve(name string) (elliptic.Curve, error) {
	switch strings.ToLower(name) {
	case "p256", "p-256":
		return elliptic.P256(), nil
	case "p384", "p-384":
		return elliptic.P384(), nil
	case "p521", "p-521":
		return elliptic.P521(), nil
	default:
		return nil, fmt.Errorf("unsupported ecdsa curve %q; use p256, p384, or p521", name)
	}
}

func rootSubject(opts CreateRootOptions) pkix.Name {
	name := pkix.Name{CommonName: opts.CommonName}
	if opts.Organization != "" {
		name.Organization = []string{opts.Organization}
	}
	if opts.OrganizationalUnit != "" {
		name.OrganizationalUnit = []string{opts.OrganizationalUnit}
	}
	if opts.Country != "" {
		name.Country = []string{opts.Country}
	}
	if opts.Province != "" {
		name.Province = []string{opts.Province}
	}
	if opts.Locality != "" {
		name.Locality = []string{opts.Locality}
	}
	return name
}

func rootKeyUsage(opts CreateRootOptions) x509.KeyUsage {
	var usage x509.KeyUsage
	if opts.AllowCertSigning {
		usage |= x509.KeyUsageCertSign
	}
	if opts.AllowCRLSigning {
		usage |= x509.KeyUsageCRLSign
	}
	return usage
}

func rootExtKeyUsage(opts CreateRootOptions) []x509.ExtKeyUsage {
	var usage []x509.ExtKeyUsage
	if opts.AllowServerAuth {
		usage = append(usage, x509.ExtKeyUsageServerAuth)
	}
	if opts.AllowClientAuth {
		usage = append(usage, x509.ExtKeyUsageClientAuth)
	}
	if opts.AllowCodeSigning {
		usage = append(usage, x509.ExtKeyUsageCodeSigning)
	}
	if opts.AllowEmailProtection {
		usage = append(usage, x509.ExtKeyUsageEmailProtection)
	}
	return usage
}

func parseIPRanges(values []string) ([]*net.IPNet, error) {
	var ranges []*net.IPNet
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		_, parsed, err := net.ParseCIDR(value)
		if err != nil {
			return nil, fmt.Errorf("invalid permitted IP range %q: %w", value, err)
		}
		ranges = append(ranges, parsed)
	}
	return ranges, nil
}

func writeRootArtifacts(opts CreateRootOptions, result *CreateRootResult, cert *x509.Certificate, key crypto.Signer, publicKey any, der, certPEM []byte) error {
	pubDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	pubPath := filepath.Join(opts.OutDir, "root-ca.pub")
	if err := os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}), 0o644); err != nil {
		return err
	}
	result.PublicKeyPath = pubPath
	if opts.WritePEM {
		pemPath := filepath.Join(opts.OutDir, "root-ca.pem")
		if err := os.WriteFile(pemPath, certPEM, 0o644); err != nil {
			return err
		}
		result.PEMPath = pemPath
	}
	if opts.WriteDER {
		derPath := filepath.Join(opts.OutDir, "root-ca.der")
		if err := os.WriteFile(derPath, der, 0o644); err != nil {
			return err
		}
		result.DERPath = derPath
	}
	metadataPath := filepath.Join(opts.OutDir, "metadata.json")
	metadata := map[string]any{
		"name":                      opts.Name,
		"subject":                   cert.Subject.String(),
		"issuer":                    cert.Issuer.String(),
		"algorithm":                 keyAlgorithm(publicKey),
		"curve":                     keyCurve(publicKey),
		"key_size":                  keySize(publicKey),
		"fingerprint_sha256":        result.Fingerprint,
		"not_before":                cert.NotBefore.Format(time.RFC3339),
		"not_after":                 cert.NotAfter.Format(time.RFC3339),
		"max_path_length":           cert.MaxPathLen,
		"offline_recommended":       opts.Offline,
		"encrypted_private_key":     opts.EncryptKey,
		"crl_enabled_future_ready":  opts.CRLEnabled,
		"ocsp_enabled_future_ready": opts.OCSPEnabled,
		"permitted_dns_domains":     opts.PermittedDNSDomains,
		"permitted_ip_ranges":       opts.PermittedIPRanges,
		"permitted_email_domains":   opts.PermittedEmailDomains,
		"key_usage":                 keyUsageNames(cert.KeyUsage),
		"extended_key_usage":        extKeyUsageNames(cert.ExtKeyUsage),
	}
	body, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(metadataPath, append(body, '\n'), 0o644); err != nil {
		return err
	}
	result.MetadataPath = metadataPath
	fingerprintsPath := filepath.Join(opts.OutDir, "fingerprints.txt")
	if err := os.WriteFile(fingerprintsPath, []byte("SHA256 Fingerprint="+result.Fingerprint+"\n"), 0o644); err != nil {
		return err
	}
	result.FingerprintsPath = fingerprintsPath
	recoveryPath := filepath.Join(opts.OutDir, "recovery.txt")
	recovery := "IronRoot Root CA recovery notes\n\n" +
		"- Keep root-ca.key encrypted and offline.\n" +
		"- Store at least two offline backups in separate locations.\n" +
		"- Do not copy root-ca.key to IronRoot servers, containers, Kubernetes, or CI.\n" +
		"- Distribute only root-ca.crt or trust-bundle contents to systems that need trust.\n"
	if opts.BackupReminder {
		recovery += "- Verify backup readability before production use.\n"
	}
	if err := os.WriteFile(recoveryPath, []byte(recovery), 0o644); err != nil {
		return err
	}
	result.RecoveryPath = recoveryPath
	if opts.GenerateTrustBundle {
		trustDir := filepath.Join(opts.OutDir, "trust-bundle")
		if err := os.MkdirAll(trustDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(trustDir, "root-ca.crt"), certPEM, 0o644); err != nil {
			return err
		}
		result.TrustBundleDir = trustDir
	}
	_ = key
	return nil
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

func keyAlgorithm(publicKey any) string {
	switch publicKey.(type) {
	case *rsa.PublicKey:
		return "rsa"
	case *ecdsa.PublicKey:
		return "ecdsa"
	default:
		return "unknown"
	}
}

func keyCurve(publicKey any) string {
	key, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return ""
	}
	switch key.Curve {
	case elliptic.P256():
		return "p256"
	case elliptic.P384():
		return "p384"
	case elliptic.P521():
		return "p521"
	default:
		return "unknown"
	}
}

func keySize(publicKey any) int {
	switch key := publicKey.(type) {
	case *rsa.PublicKey:
		return key.N.BitLen()
	case *ecdsa.PublicKey:
		return key.Curve.Params().BitSize
	default:
		return 0
	}
}

func keyUsageNames(usage x509.KeyUsage) []string {
	var out []string
	if usage&x509.KeyUsageDigitalSignature != 0 {
		out = append(out, "digital_signature")
	}
	if usage&x509.KeyUsageContentCommitment != 0 {
		out = append(out, "content_commitment")
	}
	if usage&x509.KeyUsageKeyEncipherment != 0 {
		out = append(out, "key_encipherment")
	}
	if usage&x509.KeyUsageDataEncipherment != 0 {
		out = append(out, "data_encipherment")
	}
	if usage&x509.KeyUsageKeyAgreement != 0 {
		out = append(out, "key_agreement")
	}
	if usage&x509.KeyUsageCertSign != 0 {
		out = append(out, "cert_signing")
	}
	if usage&x509.KeyUsageCRLSign != 0 {
		out = append(out, "crl_signing")
	}
	if usage&x509.KeyUsageEncipherOnly != 0 {
		out = append(out, "encipher_only")
	}
	if usage&x509.KeyUsageDecipherOnly != 0 {
		out = append(out, "decipher_only")
	}
	return out
}

func extKeyUsageNames(usages []x509.ExtKeyUsage) []string {
	var out []string
	for _, usage := range usages {
		switch usage {
		case x509.ExtKeyUsageServerAuth:
			out = append(out, "server_auth")
		case x509.ExtKeyUsageClientAuth:
			out = append(out, "client_auth")
		case x509.ExtKeyUsageCodeSigning:
			out = append(out, "code_signing")
		case x509.ExtKeyUsageEmailProtection:
			out = append(out, "email_protection")
		case x509.ExtKeyUsageTimeStamping:
			out = append(out, "timestamping")
		case x509.ExtKeyUsageOCSPSigning:
			out = append(out, "ocsp_signing")
		default:
			out = append(out, fmt.Sprintf("unknown_%d", usage))
		}
	}
	return out
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
