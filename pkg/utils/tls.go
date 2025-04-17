package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

func NewTLSConfig(caFile string) (*tls.Config, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("error reading CA certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	return &tls.Config{
		RootCAs:            certPool,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}, nil
}
