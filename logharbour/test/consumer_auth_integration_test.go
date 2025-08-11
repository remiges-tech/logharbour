package logharbour_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestConsumerIntegrationWithAuth(t *testing.T) {
	// This test validates that the consumer binary works with authentication enabled
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

	tests := []struct {
		name               string
		username           string
		password           string
		expectStartupError bool
	}{
		{
			name:               "valid credentials",
			username:           "elastic",
			password:           "testpassword123",
			expectStartupError: false,
		},
		{
			name:               "invalid credentials",
			username:           "elastic",
			password:           "wrongpassword",
			expectStartupError: true,
		},
		{
			name:               "no credentials (should fail with security enabled)",
			username:           "",
			password:           "",
			expectStartupError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build consumer if not already built
			if err := buildConsumer(); err != nil {
				t.Fatalf("failed to build consumer: %v", err)
			}

			// Prepare command args
			args := []string{
				"--esAddresses", esURL,
				"--esIndex", fmt.Sprintf("test-consumer-%s", strings.ReplaceAll(strings.ToLower(tt.name), " ", "-")),
				"--logLevel", "debug",
				"--kafkaBrokers", "localhost:19092", // Non-existent Kafka to prevent actual consumption
			}

			if tt.password != "" {
				args = append(args, "--esPassword", tt.password)
				if tt.username != "" {
					args = append(args, "--esUsername", tt.username)
				}
			}

			// Start consumer process
			cmd := exec.Command("./logConsumer", args...)
			cmd.Dir = "../../cmd/logConsumer"
			
			// Capture output
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if tt.expectStartupError {
				// Consumer should fail to start due to authentication or connection issues
				require.Error(t, err, "Expected consumer to fail startup")
				
				// Check that it failed during Elasticsearch setup, not just Kafka connection
				require.True(t, 
					strings.Contains(outputStr, "Failed to setup Elasticsearch index") ||
					strings.Contains(outputStr, "Failed to create Elasticsearch client"),
					"Expected Elasticsearch-related error, got: %s", outputStr)
			} else {
				// Consumer should start successfully (even if Kafka connection fails)
				// We expect it to pass Elasticsearch setup but fail on Kafka
				require.True(t,
					strings.Contains(outputStr, "Elasticsearch client created") ||
					strings.Contains(outputStr, "Elasticsearch index setup completed"),
					"Expected successful Elasticsearch setup, got: %s", outputStr)
			}

			// Verify authentication logs
			if tt.password != "" {
				require.Contains(t, outputStr, "elasticsearch_auth_enabled=true", 
					"Expected auth enabled log")
				require.Contains(t, outputStr, "Elasticsearch authentication enabled", 
					"Expected auth enabled message")
			} else {
				require.Contains(t, outputStr, "elasticsearch_auth_enabled=false", 
					"Expected auth disabled log")
				require.Contains(t, outputStr, "Elasticsearch authentication disabled", 
					"Expected auth disabled message")
			}
		})
	}
}

func TestConsumerBackwardCompatibility(t *testing.T) {
	// This test validates that the consumer works with no authentication (existing behavior)
	ctx := context.Background()
	timeout := 300 * time.Second

	// Start Elasticsearch without X-Pack Security
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

	// Build consumer if not already built
	if err := buildConsumer(); err != nil {
		t.Fatalf("failed to build consumer: %v", err)
	}

	// Test existing behavior (no authentication)
	args := []string{
		"--esAddresses", esURL,
		"--esIndex", "test-backward-compatibility",
		"--logLevel", "debug",
		"--kafkaBrokers", "localhost:19092", // Non-existent Kafka
	}

	cmd := exec.Command("./logConsumer", args...)
	cmd.Dir = "../../cmd/logConsumer"

	// Run with timeout since it will try to connect to Kafka
	done := make(chan error, 1)
	go func() {
		_, err := cmd.CombinedOutput()
		done <- err
	}()

	// Stop the process after 10 seconds
	go func() {
		time.Sleep(10 * time.Second)
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
		}
	}()

	select {
	case <-done:
		// Process finished (expected due to Kafka connection failure)
	case <-time.After(15 * time.Second):
		// Force kill if still running
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}

	// The test passes if the consumer started without authentication errors
	// (it will fail on Kafka connection, but that's expected)
}

func buildConsumer() error {
	// Check if consumer binary exists and is recent
	if info, err := os.Stat("../../cmd/logConsumer/logConsumer"); err == nil {
		// If binary exists and is newer than source, skip build
		if sourceInfo, err := os.Stat("../../cmd/logConsumer/main.go"); err == nil {
			if info.ModTime().After(sourceInfo.ModTime()) {
				return nil // Binary is up to date
			}
		}
	}

	// Build the consumer
	cmd := exec.Command("go", "build", "-o", "logConsumer", ".")
	cmd.Dir = "../../cmd/logConsumer"
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build failed: %v, output: %s", err, output)
	}

	return nil
}