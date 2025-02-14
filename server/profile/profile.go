package profile

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/usememos/memos/server/version"
)

type Feature string

// Profile is the configuration to start main server.
type Profile struct {
	// Mode can be "prod" or "dev" or "demo"
	Mode string `json:"mode"`
	// Port is the binding port for server
	Port int `json:"-"`
	// Data is the data directory
	Data string `json:"-"`
	// DSN points to where Memos stores its own data
	DSN string `json:"-"`
	// Version is the current version of server
	Version string `json:"version"`
	// Feat is the feature set of server, split by comma
	Feat string `json:"feature"`
}

const (
	FeatStorageLocal Feature = "STORAGE_LOCAL"
	FeatSSO          Feature = "SSO"
	FeatStorageS3    Feature = "STORAGE_S3"
)

func (p *Profile) IsFeatEnabled(feat Feature) bool {
	for _, f := range strings.Split(p.Feat, ",") {
		if f == string(feat) {
			return true
		}
	}
	return p.Feat == "ALL"
}

func checkDSN(dataDir string) (string, error) {
	// Convert to absolute path if relative path is supplied.
	if !filepath.IsAbs(dataDir) {
		absDir, err := filepath.Abs(filepath.Dir(os.Args[0]) + "/" + dataDir)
		if err != nil {
			return "", err
		}
		dataDir = absDir
	}

	// Trim trailing / in case user supplies
	dataDir = strings.TrimRight(dataDir, "/")

	if _, err := os.Stat(dataDir); err != nil {
		return "", fmt.Errorf("unable to access data folder %s, err %w", dataDir, err)
	}

	return dataDir, nil
}

// GetDevProfile will return a profile for dev or prod.
func GetProfile() (*Profile, error) {
	profile := Profile{}
	err := viper.Unmarshal(&profile)
	if err != nil {
		return nil, err
	}

	if profile.Mode != "demo" && profile.Mode != "dev" && profile.Mode != "prod" {
		profile.Mode = "demo"
	}

	// Make feat upper
	profile.Feat = strings.ToUpper(profile.Feat)

	// If no feature flags are supplied, use all features when not in prod mode.
	if profile.Feat == "" && profile.Mode != "prod" {
		profile.Feat = "ALL"
	}

	if profile.Mode == "prod" && profile.Data == "" {
		profile.Data = "/var/opt/memos"
	}

	profile.Data, err = filepath.Abs(profile.Data)
	if err != nil {
		return nil, err
	}

	// mkdir Data/resources
	if profile.IsFeatEnabled(FeatStorageLocal) {
		err = os.MkdirAll(path.Join(profile.Data, "resources"), 0755)
		if err != nil {
			return nil, err
		}
	}

	dataDir, err := checkDSN(profile.Data)
	if err != nil {
		fmt.Printf("Failed to check dsn: %s, err: %+v\n", dataDir, err)
		return nil, err
	}

	profile.Data = dataDir
	profile.DSN = fmt.Sprintf("%s/memos_%s.db", dataDir, profile.Mode)
	profile.Version = version.GetCurrentVersion(profile.Mode)

	return &profile, nil
}
