package logharbour_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/remiges-tech/logharbour/logharbour"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestElasticsearchBasicAuthWithContainer(t *testing.T) {
	ctx := context.Background()
	timeout := 300 * time.Second

	tests := []struct {
		name           string
		username       string
		password       string
		expectAuthFail bool
	}{
		{
			name:           "valid credentials",
			username:       "elastic",
			password:       "testpassword123",
			expectAuthFail: false,
		},
		{
			name:           "invalid password",
			username:       "elastic",
			password:       "wrongpassword",
			expectAuthFail: true,
		},
		{
			name:           "invalid username",
			username:       "wronguser",
			password:       "testpassword123",
			expectAuthFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create container request with X-Pack Security enabled (free tier)
			req := testcontainers.ContainerRequest{
				Image:        "docker.elastic.co/elasticsearch/elasticsearch:8.12.0",
				ExposedPorts: []string{"9200/tcp"},
				Env: map[string]string{
					"discovery.type":                        "single-node",
					"xpack.security.enabled":                "true",
					"ELASTIC_PASSWORD":                      "testpassword123",
					"xpack.security.transport.ssl.enabled": "false", // Simplified for testing
					"xpack.security.http.ssl.enabled":      "false", // Use HTTP for testing
				},
				WaitingFor: wait.ForAll(
					wait.ForHTTP("/_cluster/health").
						WithPort("9200").
						WithBasicAuth("elastic", "testpassword123").
						WithStartupTimeout(timeout),
					wait.ForLog("started").WithStartupTimeout(timeout),
				),
			}

			// Start container
			container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
			})
			if err != nil {
				t.Fatalf("failed to start container: %v", err)
			}
			defer func() {
				if err := container.Terminate(ctx); err != nil {
					t.Logf("failed to terminate container: %v", err)
				}
			}()

			// Get container host and port
			host, err := container.Host(ctx)
			require.NoError(t, err)

			port, err := container.MappedPort(ctx, "9200")
			require.NoError(t, err)

			esURL := fmt.Sprintf("http://%s:%s", host, port.Port())

			// Test with provided credentials
			config := elasticsearch.Config{
				Addresses: []string{esURL},
				Username:  tt.username,
				Password:  tt.password,
			}

			client, err := logharbour.NewElasticsearchClient(config)
			require.NoError(t, err)
			require.NotNil(t, client)

			// Test connection by performing actual operations
			indexName := fmt.Sprintf("test-index-%s", strings.ReplaceAll(strings.ToLower(tt.name), " ", "-"))
			err = testElasticsearchConnection(client, indexName)
			
			if tt.expectAuthFail {
				require.Error(t, err, "Expected authentication to fail with invalid credentials")
			} else {
				require.NoError(t, err, "Expected authentication to succeed with valid credentials")
			}
		})
	}
}

func TestElasticsearchNoAuthWhenSecurityEnabled(t *testing.T) {
	ctx := context.Background()
	timeout := 300 * time.Second

	// Start Elasticsearch with X-Pack Security enabled
	req := testcontainers.ContainerRequest{
		Image:        "docker.elastic.co/elasticsearch/elasticsearch:8.12.0",
		ExposedPorts: []string{"9200/tcp"},
		Env: map[string]string{
			"discovery.type":                        "single-node",
			"xpack.security.enabled":                "true",
			"ELASTIC_PASSWORD":                      "testpassword123",
			"xpack.security.transport.ssl.enabled": "false",
			"xpack.security.http.ssl.enabled":      "false",
		},
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/_cluster/health").
				WithPort("9200").
				WithBasicAuth("elastic", "testpassword123").
				WithStartupTimeout(timeout),
			wait.ForLog("started").WithStartupTimeout(timeout),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "9200")
	require.NoError(t, err)

	esURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Test with no credentials when security is enabled
	noAuthConfig := elasticsearch.Config{
		Addresses: []string{esURL},
		// No username/password provided
	}

	noAuthClient, err := logharbour.NewElasticsearchClient(noAuthConfig)
	require.NoError(t, err) // Client creation should succeed
	require.NotNil(t, noAuthClient)

	// Operations should fail without authentication
	err = testElasticsearchConnection(noAuthClient, "test-index-noauth")
	require.Error(t, err, "Expected operations to fail without authentication when security is enabled")
}

func TestElasticsearchBackwardCompatibility(t *testing.T) {
	ctx := context.Background()
	timeout := 300 * time.Second

	// Start Elasticsearch without X-Pack Security (like current LogHarbour setup)
	req := testcontainers.ContainerRequest{
		Image:        "docker.elastic.co/elasticsearch/elasticsearch:8.12.0",
		ExposedPorts: []string{"9200/tcp"},
		Env: map[string]string{
			"discovery.type":         "single-node",
			"xpack.security.enabled": "false", // Security disabled
		},
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/_cluster/health").
				WithPort("9200").
				WithStartupTimeout(timeout),
			wait.ForLog("started").WithStartupTimeout(timeout),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "9200")
	require.NoError(t, err)

	esURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Test that existing configuration (no auth) still works
	config := elasticsearch.Config{
		Addresses: []string{esURL},
		// No authentication configured
	}

	client, err := logharbour.NewElasticsearchClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Operations should succeed without authentication when security is disabled
	err = testElasticsearchConnection(client, "test-index-backward-compat")
	require.NoError(t, err, "Expected operations to succeed without auth when security is disabled")
}

func TestElasticsearchHTTPSWithBasicAuth(t *testing.T) {
	ctx := context.Background()
	timeout := 300 * time.Second

	// Start Elasticsearch with HTTPS enabled
	req := testcontainers.ContainerRequest{
		Image:        "docker.elastic.co/elasticsearch/elasticsearch:8.12.0",
		ExposedPorts: []string{"9200/tcp"},
		Env: map[string]string{
			"discovery.type":                       "single-node",
			"xpack.security.enabled":               "true",
			"ELASTIC_PASSWORD":                     "testpassword123",
			"xpack.security.transport.ssl.enabled": "false",
			"xpack.security.http.ssl.enabled":     "true",
			"xpack.security.http.ssl.keystore.type": "PKCS12",
		},
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/_cluster/health").
				WithPort("9200").
				WithTLS(true, nil). // Accept any certificate for testing
				WithBasicAuth("elastic", "testpassword123").
				WithStartupTimeout(timeout),
			wait.ForLog("started").WithStartupTimeout(timeout),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "9200")
	require.NoError(t, err)

	esURL := fmt.Sprintf("https://%s:%s", host, port.Port())

	// Test HTTPS connection with basic auth and certificate verification disabled
	config := elasticsearch.Config{
		Addresses: []string{esURL},
		Username:  "elastic",
		Password:  "testpassword123",
		Transport: logharbour.InsecureTransport(), // For testing only
	}

	client, err := logharbour.NewElasticsearchClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Test connection over HTTPS
	err = testElasticsearchConnection(client, "test-index-https")
	require.NoError(t, err, "Expected HTTPS connection with basic auth to succeed")
}

func TestElasticsearchConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      elasticsearch.Config
		expectError bool
	}{
		{
			name: "valid basic auth config",
			config: elasticsearch.Config{
				Addresses: []string{"http://localhost:9200"},
				Username:  "elastic",
				Password:  "password",
			},
			expectError: false,
		},
		{
			name: "valid no auth config",
			config: elasticsearch.Config{
				Addresses: []string{"http://localhost:9200"},
			},
			expectError: false,
		},
		{
			name: "empty addresses",
			config: elasticsearch.Config{
				Addresses: []string{},
				Username:  "elastic",
				Password:  "password",
			},
			expectError: false, // Client creation succeeds, connection will fail at runtime
		},
		{
			name: "username without password",
			config: elasticsearch.Config{
				Addresses: []string{"http://localhost:9200"},
				Username:  "elastic",
				Password:  "", // Empty password
			},
			expectError: false, // Client creation succeeds, auth will fail at runtime
		},
		{
			name: "password without username",
			config: elasticsearch.Config{
				Addresses: []string{"http://localhost:9200"},
				Username:  "", // Empty username
				Password:  "password",
			},
			expectError: false, // Client creation succeeds, auth will fail at runtime
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := logharbour.NewElasticsearchClient(tt.config)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
			}
		})
	}
}

// testElasticsearchConnection verifies authentication by performing actual operations
func testElasticsearchConnection(client *logharbour.ElasticsearchClient, indexName string) error {
	// Test connection by trying to create an index
	mapping := `{
		"mappings": {
			"properties": {
				"message": {
					"type": "text"
				},
				"timestamp": {
					"type": "date"
				}
			}
		}
	}`

	err := client.CreateIndex(indexName, mapping)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Test write operation
	testDoc := logharbour.BulkDocument{
		ID:   "test-1",
		Body: `{"message": "test message", "timestamp": "2024-01-01T00:00:00Z"}`,
	}

	result, err := client.BulkWrite(indexName, []logharbour.BulkDocument{testDoc})
	if err != nil {
		return fmt.Errorf("failed to write document: %w", err)
	}

	if result.Failed > 0 {
		return fmt.Errorf("document write failed: %d errors", result.Failed)
	}

	return nil
}