package preflight

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/v2/require"
	"github.com/kvick-org/pkg/version"
)

func TestCheck(t *testing.T) {
	versionInfo := version.Info{
		Runtime: version.Runtime{
			OS:     "linux",
			Distro: "ubuntu",
		},
	}
	err := Check(versionInfo)
	require.NoError(t, err)

	t.Setenv(KubernetesEnvVar, "foo")
	err = Check(versionInfo)
	require.EqualError(t, err, "unsupported container distro ubuntu")

	versionInfo.Runtime.Distro = "Distroless"
	err = Check(versionInfo)
	require.NoError(t, err)
}

func TestCheckContainerdConfig(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty path",
			configPath:  "",
			expectError: false,
		},
		{
			name:        "config file not found",
			configPath:  "/nonexistent/path/to/config.toml",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "config file with discard_unpacked_layers = true (v1)",
			configPath:  "",
			expectError: false,
		},
		{
			name:        "config file with discard_unpacked_layers = false (v1)",
			configPath:  "",
			expectError: true,
			errorMsg:    "does not have discard_unpacked_layers = true",
		},
		{
			name:        "config file missing key (v1)",
			configPath:  "",
			expectError: true,
			errorMsg:    "is missing discard_unpacked_layers configuration",
		},
		{
			name:        "config file with v2 key (io.containerd.cri.v1.images.discard_unpacked_layers = true)",
			configPath:  "",
			expectError: false,
		},
		{
			name:        "config file with v2 key = false",
			configPath:  "",
			expectError: true,
			errorMsg:    "does not have discard_unpacked_layers = true",
		},
		{
			name:        "config file with nested v1 structure",
			configPath:  "",
			expectError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file for tests that need it
			if tt.configPath == "" && tt.expectError {
				tt.configPath = createTestConfig(t, tt.name)
			}

			err := CheckContainerdConfig(tt.configPath)
			if tt.expectError {
				require.Error(t, err, "expected error for test case %s", tt.name)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "did not expect error for test case %s", tt.name)
			}
		})
	}
}

func createTestConfig(t *testing.T, testType string) string {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	var configContent string
	switch testType {
	case "config file with discard_unpacked_layers = true (v1)":
		configContent = `[plugins."io.containerd.grpc.v1.cri"]
  [plugins."io.containerd.grpc.v1.cri".containerd]
    discard_unpacked_layers = true
`
	case "config file with discard_unpacked_layers = false (v1)":
		configContent = `[plugins."io.containerd.grpc.v1.cri"]
  [plugins."io.containerd.grpc.v1.cri".containerd]
    discard_unpacked_layers = false
`
	case "config file missing key (v1)":
		configContent = `[plugins."io.containerd.grpc.v1.cri"]
  [plugins."io.containerd.grpc.v1.cri".containerd]
    sandbox_mode = "pods"
`
	case "config file with v2 key (io.containerd.cri.v1.images.discard_unpacked_layers = true)":
		configContent = `[plugins."io.containerd.cri.v1.images"]
  discard_unpacked_layers = true
`
	case "config file with v2 key = false":
		configContent = `[plugins."io.containerd.cri.v1.images"]
  discard_unpacked_layers = false
`
	case "config file with nested v1 structure":
		configContent = `[plugins."io.containerd.grpc.v1.cri"]
  [plugins."io.containerd.grpc.v1.cri".containerd]
    [plugins."io.containerd.grpc.v1.cri".containerd.runc]
      runtime_type = "io.containerd.runc.v2"
`
	default:
		t.Fatalf("unknown test type: %s", testType)
	}

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	return configPath
}
