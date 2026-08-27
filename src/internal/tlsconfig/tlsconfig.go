package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Build creates a client-side TLS configuration from the certificate settings
// shared by service SDKs and readiness checks.
func Build(service, caFile, certFile, keyFile, serverName string) (*tls.Config, error) {
	if caFile == "" && certFile == "" && keyFile == "" && serverName == "" {
		return nil, nil
	}
	if (certFile == "") != (keyFile == "") {
		return nil, fmt.Errorf("%s TLS client certificate and key must be configured together", service)
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName}
	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read %s CA certificate: %w", service, err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("%s TLS CA certificate file contains no valid certificates", service)
		}
		tlsConfig.RootCAs = roots
	}
	if certFile != "" {
		certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load %s client certificate: %w", service, err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}
	return tlsConfig, nil
}

// HTTPClient builds an HTTP client that uses the supplied TLS configuration.
func HTTPClient(service, caFile, certFile, keyFile, serverName string, timeout time.Duration) (*http.Client, error) {
	tlsConfig, err := Build(service, caFile, certFile, keyFile, serverName)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	return &http.Client{Transport: transport, Timeout: timeout}, nil
}
