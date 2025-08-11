package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCreateTLSTransport tests the createTLSTransport function with various certificate scenarios.
// 
// Testing Scope:
// This test validates the certificate loading and parsing infrastructure only.
// It verifies that our custom certificate loading logic can:
// 1. Read CA certificate files from disk
// 2. Parse PEM-encoded certificates correctly  
// 3. Create a custom HTTP transport with the certificate pool
// 4. Handle invalid certificate data appropriately
// 5. Handle missing certificate files gracefully
//
// Testing Limitations:
// This test does NOT validate actual TLS handshake behavior because:
// 1. We create a self-signed CA certificate, not a proper server certificate signed by that CA
// 2. No actual network connection or TLS handshake is performed
// 3. The test relies on Go's standard library crypto/tls for actual TLS validation
//
// For end-to-end TLS validation, see the integration tests in auth_test.go which use
// testcontainers with real Elasticsearch instances over HTTPS connections.
func TestCreateTLSTransport(t *testing.T) {
	tests := []struct {
		name        string
		setupCert   func(t *testing.T) string
		expectError bool
	}{
		{
			name: "valid CA certificate",
			setupCert: func(t *testing.T) string {
				return createTestCACertificate(t)
			},
			expectError: false,
		},
		{
			name: "invalid certificate file",
			setupCert: func(t *testing.T) string {
				return createInvalidCertificate(t)
			},
			expectError: true,
		},
		{
			name: "non-existent file",
			setupCert: func(t *testing.T) string {
				return "/nonexistent/path/to/cert.pem"
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certPath := tt.setupCert(t)
			
			// Clean up file if it exists
			if _, err := os.Stat(certPath); err == nil {
				defer os.Remove(certPath)
			}

			transport, err := createTLSTransport(certPath)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, transport)
			} else {
				require.NoError(t, err)
				require.NotNil(t, transport)
				require.NotNil(t, transport.TLSClientConfig)
				require.NotNil(t, transport.TLSClientConfig.RootCAs)
			}
		})
	}
}

// createTestCACertificate generates a self-signed CA certificate for testing certificate loading.
// Note: This creates a CA certificate that signs itself (self-signed), not a server certificate
// signed by a CA. This is sufficient for testing certificate parsing but not actual TLS validation.
func createTestCACertificate(t *testing.T) string {
	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test City"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Create temporary file
	tmpFile, err := ioutil.TempFile("", "test_ca_*.pem")
	require.NoError(t, err)
	defer tmpFile.Close()

	// Write certificate to file in PEM format
	err = pem.Encode(tmpFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	require.NoError(t, err)

	return tmpFile.Name()
}

// createInvalidCertificate creates a file with invalid certificate content for testing error handling.
func createInvalidCertificate(t *testing.T) string {
	// Create temporary file with invalid content
	tmpFile, err := ioutil.TempFile("", "invalid_cert_*.pem")
	require.NoError(t, err)
	defer tmpFile.Close()

	// Write invalid content
	_, err = tmpFile.WriteString("This is not a valid certificate")
	require.NoError(t, err)

	return tmpFile.Name()
}

// TestElasticsearchClientCreationWithCerts tests the full client creation flow with CA certificates.
// This validates that createElasticsearchClient correctly integrates the TLS transport from createTLSTransport.
func TestElasticsearchClientCreationWithCerts(t *testing.T) {
	// Initialize logger for the test
	setupLogger("info")
	
	// Test the full createElasticsearchClient function with CA certificate
	certPath := createTestCACertificate(t)
	defer os.Remove(certPath)

	addresses := "https://localhost:9200"
	username := "elastic"
	password := "testpassword"

	client, err := createElasticsearchClient(addresses, username, password, certPath)
	
	// Client creation should succeed even though we can't connect
	require.NoError(t, err)
	require.NotNil(t, client)
}

// TestElasticsearchClientCreationNoCerts tests client creation without CA certificates (backward compatibility).
func TestElasticsearchClientCreationNoCerts(t *testing.T) {
	// Initialize logger for the test
	setupLogger("info")
	
	// Test the full createElasticsearchClient function without CA certificate
	addresses := "http://localhost:9200"
	username := "elastic" 
	password := "testpassword"
	caCertPath := "" // No certificate

	client, err := createElasticsearchClient(addresses, username, password, caCertPath)
	
	require.NoError(t, err)
	require.NotNil(t, client)
}