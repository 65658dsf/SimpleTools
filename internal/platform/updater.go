package platform

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type UpdateSource interface {
	Check(current string) (*UpdateInfo, error)
	DownloadAndInstall(*UpdateInfo) error
}

type UpdateAsset struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	URL          string `json:"url"`
	SHA256       string `json:"sha256"`
	Signature    string `json:"signature"`
	Size         int64  `json:"size,omitempty"`
}

type UpdateInfo struct {
	Available bool          `json:"available"`
	Version   string        `json:"version"`
	URL       string        `json:"url,omitempty"`
	Notes     string        `json:"notes,omitempty"`
	AssetID   string        `json:"assetId,omitempty"`
	SHA256    string        `json:"sha256,omitempty"`
	Signature string        `json:"signature,omitempty"`
	Assets    []UpdateAsset `json:"assets,omitempty"`
}

type GitHubUpdater struct {
	Owner       string
	Repository  string
	ManifestURL string
	PublicKey   ed25519.PublicKey
	Client      *http.Client
}

func (u *GitHubUpdater) SetPublicKey(encoded string) error {
	if strings.TrimSpace(encoded) == "" {
		u.PublicKey = nil
		return nil
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("invalid update public key")
	}
	u.PublicKey = ed25519.PublicKey(key)
	return nil
}

func NewGitHubUpdater(owner, repository string) *GitHubUpdater {
	return &GitHubUpdater{Owner: owner, Repository: repository, Client: &http.Client{Timeout: 15 * time.Second}}
}

type NoopUpdater struct{}

func (NoopUpdater) Check(string) (*UpdateInfo, error) { return &UpdateInfo{}, nil }
func (NoopUpdater) DownloadAndInstall(*UpdateInfo) error {
	return errors.New("automatic updates are not configured")
}

func (u *GitHubUpdater) manifestURL() string {
	if u.ManifestURL != "" {
		return u.ManifestURL
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/update-manifest.json", u.Owner, u.Repository)
}

func (u *GitHubUpdater) Check(current string) (*UpdateInfo, error) {
	if strings.TrimSpace(u.Owner) == "" || strings.TrimSpace(u.Repository) == "" {
		return &UpdateInfo{}, nil
	}
	client := u.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(u.manifestURL())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update manifest returned HTTP %d", resp.StatusCode)
	}
	var manifest struct {
		Version string        `json:"version"`
		Notes   string        `json:"notes"`
		Assets  []UpdateAsset `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode update manifest: %w", err)
	}
	if compareVersions(manifest.Version, current) <= 0 {
		return &UpdateInfo{Version: manifest.Version, Notes: manifest.Notes, Assets: manifest.Assets}, nil
	}
	asset, ok := selectAsset(manifest.Assets, runtime.GOOS, runtime.GOARCH)
	if !ok {
		return &UpdateInfo{Available: true, Version: manifest.Version, Notes: manifest.Notes, Assets: manifest.Assets}, nil
	}
	return &UpdateInfo{Available: true, Version: manifest.Version, URL: asset.URL, Notes: manifest.Notes, AssetID: asset.ID, SHA256: asset.SHA256, Signature: asset.Signature, Assets: manifest.Assets}, nil
}

func selectAsset(assets []UpdateAsset, goos, goarch string) (UpdateAsset, bool) {
	for _, asset := range assets {
		if strings.EqualFold(asset.Platform, goos) && strings.EqualFold(asset.Architecture, goarch) {
			return asset, true
		}
	}
	return UpdateAsset{}, false
}

func (u *GitHubUpdater) DownloadAndInstall(info *UpdateInfo) error {
	if info == nil || !info.Available || info.URL == "" {
		return errors.New("no verified update is available")
	}
	if strings.TrimSpace(info.SHA256) == "" {
		return errors.New("update manifest is missing a SHA-256 checksum")
	}
	if strings.TrimSpace(info.Signature) == "" {
		return errors.New("update manifest is missing an Ed25519 signature")
	}
	client := u.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(info.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	cacheDir := UpdateCachePath()
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return err
	}
	// Keep the installer on disk after a successful launch. Removing it
	// immediately races with the platform installer opening it.
	// Windows uses the file extension to decide whether a path is an
	// executable. Keep the installer suffix on the downloaded temporary file;
	// without it CreateProcess reports "executable file not found in %PATH%".
	tmp, err := os.CreateTemp(cacheDir, updateTempPattern(runtime.GOOS))
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, io.LimitReader(resp.Body, 1<<31)); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if info.SHA256 != "" {
		if err := verifySHA256(tmpPath, info.SHA256); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
	}
	if info.Signature != "" {
		if len(u.PublicKey) != ed25519.PublicKeySize {
			_ = os.Remove(tmpPath)
			return errors.New("update signature cannot be verified: public key is not configured")
		}
		if err := verifySignature(tmpPath, info.Signature, u.PublicKey); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
	}
	if err := launchInstaller(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func verifySHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(strings.TrimSpace(expected), actual) {
		return fmt.Errorf("update checksum mismatch: got %s", actual)
	}
	return nil
}

func verifySignature(path, encoded string, key ed25519.PublicKey) error {
	f, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	signature, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || !ed25519.Verify(key, f, signature) {
		return errors.New("update signature verification failed")
	}
	return nil
}

// updateTempPattern returns a temporary-file pattern with the extension
// expected by the platform installer. os.CreateTemp appends the random
// component at the '*' marker, so the resulting path remains unique while
// retaining a launchable suffix.
func updateTempPattern(goos string) string {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "windows":
		return "simpletools-update-*.exe"
	case "darwin":
		return "simpletools-update-*.dmg"
	default:
		return "simpletools-update-*"
	}
}

func compareVersions(a, b string) int {
	parse := func(value string) []int {
		value = strings.TrimPrefix(strings.TrimSpace(value), "v")
		parts := strings.Split(value, ".")
		out := make([]int, len(parts))
		for i, part := range parts {
			out[i], _ = strconv.Atoi(part)
		}
		return out
	}
	left, right := parse(a), parse(b)
	for i := 0; i < len(left) || i < len(right); i++ {
		lv, rv := 0, 0
		if i < len(left) {
			lv = left[i]
		}
		if i < len(right) {
			rv = right[i]
		}
		if lv < rv {
			return -1
		}
		if lv > rv {
			return 1
		}
	}
	return 0
}

func UpdateManifestJSON(version, notes string, assets []UpdateAsset) ([]byte, error) {
	return json.MarshalIndent(struct {
		Version string        `json:"version"`
		Notes   string        `json:"notes,omitempty"`
		Assets  []UpdateAsset `json:"assets"`
	}{Version: version, Notes: notes, Assets: assets}, "", "  ")
}

func UpdateCachePath() string { return filepath.Join(os.TempDir(), "simpletools-update") }
