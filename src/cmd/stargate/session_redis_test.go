package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	rediskitclient "github.com/soulteary/redis-kit/client"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

// redisTestCertificate returns a self-signed server certificate for
// 127.0.0.1 and "redis.test", and the path of a PEM file that holds it.
func redisTestCertificate(t *testing.T) (tls.Certificate, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "redis.test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"redis.test"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	caFile := filepath.Join(t.TempDir(), "redis-ca.pem")
	require.NoError(t, os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, caFile
}

func setSessionRedisOptions(t *testing.T, addr, username, password, tlsEnabled, caFile, serverName string) {
	t.Helper()
	for variable, value := range map[*config.EnvVariable]string{
		&config.SessionStorageRedisAddr:          addr,
		&config.SessionStorageRedisUsername:      username,
		&config.SessionStorageRedisPassword:      password,
		&config.SessionStorageRedisTLSEnabled:    tlsEnabled,
		&config.SessionStorageRedisTLSCACertFile: caFile,
		&config.SessionStorageRedisTLSServerName: serverName,
	} {
		original := variable.Value
		t.Cleanup(func() { variable.Value = original })
		variable.Value = value
	}
}

// A Redis that requires TLS and an ACL user is reachable only when both
// options reach the client.
func TestSessionRedisConfigConnectsWithTLSAndACLUser(t *testing.T) {
	certificate, caFile := redisTestCertificate(t)
	server, err := miniredis.RunTLS(&tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	require.NoError(t, err)
	t.Cleanup(server.Close)
	server.RequireUserAuth("stargate", "redis-secret")

	setSessionRedisOptions(t, server.Addr(), "stargate", "redis-secret", "true", caFile, "")
	cfg, err := sessionRedisConfig(0)
	require.NoError(t, err)
	require.Equal(t, "stargate", cfg.Username)
	require.NotNil(t, cfg.TLSConfig)
	require.False(t, cfg.TLSConfig.InsecureSkipVerify)
	require.GreaterOrEqual(t, cfg.TLSConfig.MinVersion, uint16(tls.VersionTLS12))

	client, err := rediskitclient.NewClient(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Set(context.Background(), "stargate:session:probe", "1", time.Minute).Err())
	require.True(t, server.Exists("stargate:session:probe"))
}

func TestSessionRedisConfigVerifiesTheServerCertificate(t *testing.T) {
	certificate, caFile := redisTestCertificate(t)
	server, err := miniredis.RunTLS(&tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	require.NoError(t, err)
	t.Cleanup(server.Close)

	connect := func(caFile, serverName string) error {
		setSessionRedisOptions(t, server.Addr(), "", "", "true", caFile, serverName)
		cfg, err := sessionRedisConfig(0)
		require.NoError(t, err)
		cfg.MaxRetries = -1
		client, err := rediskitclient.NewClient(cfg)
		if err == nil {
			_ = client.Close()
		}
		return err
	}

	// The certificate is not signed by a system root.
	require.Error(t, connect("", ""))
	// The CA is trusted but the certificate does not carry this name.
	require.Error(t, connect(caFile, "other.test"))
	// The server name replaces the address for verification.
	require.NoError(t, connect(caFile, "redis.test"))
}

func TestSessionRedisConfigStaysPlaintextByDefault(t *testing.T) {
	setSessionRedisOptions(t, "redis.internal:6379", "", "redis-secret", "false", "", "")

	cfg, err := sessionRedisConfig(3)
	require.NoError(t, err)
	require.Nil(t, cfg.TLSConfig)
	require.Empty(t, cfg.Username)
	require.Equal(t, "redis.internal:6379", cfg.Addr)
	require.Equal(t, "redis-secret", cfg.Password)
	require.Equal(t, 3, cfg.DB)
	require.Equal(t, 5*time.Second, cfg.DialTimeout)
}

func TestSessionRedisConfigReportsUnreadableCAFile(t *testing.T) {
	setSessionRedisOptions(t, "redis.internal:6379", "", "", "true", filepath.Join(t.TempDir(), "missing.pem"), "")

	_, err := sessionRedisConfig(0)
	require.ErrorContains(t, err, "Redis CA certificate")
}
