package preflight

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kvick-org/pkg/version"
	"github.com/pelletier/go-toml/v2"
)

const (
	KubernetesEnvVar            = "KUBERNETES_SERVICE_HOST"
	ContainerdConfigPath        = "CONTAINERD_CONFIG_PATH"
	DefaultContainerdConfigPath = "/etc/containerd/config.toml"
)

type containerdConfig struct {
	Root    string         `toml:"root"`
	Plugins map[string]any `toml:"plugins"`
}

func Check(versionInfo version.Info) error {
	if os.Getenv(KubernetesEnvVar) == "" {
		return nil
	}
	if versionInfo.Runtime.OS == "linux" && versionInfo.Runtime.Distro == "Distroless" {
		return nil
	}
	return fmt.Errorf("unsupported container distro %s", versionInfo.Runtime.Distro)
}

func CheckContainerdConfig(configPath string) error {
	if configPath == "" {
		return nil
	}
	configPath = filepath.Clean(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("containerd config file not found at %s; please mount the config file or set a valid path in containerdConfigPath", configPath)
		}
		return fmt.Errorf("failed to read containerd config at %s: %w", configPath, err)
	}

	var cfg containerdConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse containerd config at %s: %w", configPath, err)
	}

	// Check v1 config format: [plugins."io.containerd.grpc.v1.cri"]
	if plugins, ok := cfg.Plugins["io.containerd.grpc.v1.cri"].(map[string]any); ok {
		if containerd, ok := plugins["containerd"].(map[string]any); ok {
			if discardLayers, ok := containerd["discard_unpacked_layers"].(any); ok {
				if discardLayers == true {
					return nil
				}
				// discard_layers is false or not set
				return fmt.Errorf("containerd config at %s does not have discard_unpacked_layers = true", configPath)
			}
			// discard_layers key doesn't exist
			return fmt.Errorf("containerd config at %s is missing discard_unpacked_layers configuration", configPath)
		}
	}

	// Check v2 config format: [plugins."io.containerd.cri.v1.images"]
	if plugins, ok := cfg.Plugins["io.containerd.cri.v1.images"].(map[string]any); ok {
		if discardLayers, ok := plugins["discard_unpacked_layers"].(any); ok {
			if discardLayers == true {
				return nil
			}
			// discard_layers is false or not set
			return fmt.Errorf("containerd config at %s does not have discard_unpacked_layers = true", configPath)
		}
		// discard_layers key doesn't exist
		return fmt.Errorf("containerd config at %s is missing discard_unpacked_layers configuration", configPath)
	}

	// Neither v1 nor v2 plugins section found
	return fmt.Errorf("containerd config at %s does not have discard_unpacked_layers = true", configPath)
}
