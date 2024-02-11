package quicpunch

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

func makeCert(names []string, caRoot []byte, caPrivateKey crypto.PrivateKey) (tls.Certificate, error) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Local"},
		},
		DNSNames:  names,
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),

		// these might get overridden if we have a CA to sign with
		IsCA:     true,
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,

		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	caCert := &template
	var signingKey crypto.PrivateKey = pk
	if len(caRoot) > 0 {
		template.IsCA = false
		template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
		caCert, err = x509.ParseCertificate(caRoot)
		if err != nil {
			return tls.Certificate{}, err
		}
		signingKey = caPrivateKey
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, &pk.PublicKey, signingKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	var cert tls.Certificate
	cert.Certificate = append(cert.Certificate, derBytes)
	cert.PrivateKey = pk

	return cert, nil
}
