package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bren-wp/Host-FTP/internal/model"
)

const (
	profileBundleSchema   = 1
	maxProfileBundleBytes = 1 << 20
	maxImportedProfiles   = 500
)

type profileBundle struct {
	Schema     int                    `json:"schema"`
	Product    string                 `json:"product"`
	ExportedAt string                 `json:"exported_at"`
	Profiles   []profileBundleProfile `json:"profiles"`
}

type profileBundleProfile struct {
	Name           string `json:"name"`
	Protocol       string `json:"protocol"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	PrivateKeyPath string `json:"private_key_path,omitempty"`
	RemotePath     string `json:"remote_path,omitempty"`
	LocalPath      string `json:"local_path,omitempty"`
}

type ProfileImportResult struct {
	Imported int
	Skipped  int
}

func exportedProfile(profile model.PublicProfile) profileBundleProfile {
	return profileBundleProfile{
		Name:           profile.Name,
		Protocol:       profile.Protocol,
		Host:           profile.Host,
		Port:           profile.Port,
		Username:       profile.Username,
		PrivateKeyPath: profile.PrivateKeyPath,
		RemotePath:     profile.RemotePath,
		LocalPath:      profile.LocalPath,
	}
}

func (e *Engine) ExportProfiles(path string) (int, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, errors.New("export path is empty")
	}
	profiles, err := e.profiles.List()
	if err != nil {
		return 0, err
	}
	bundle := profileBundle{
		Schema:     profileBundleSchema,
		Product:    "Ghost FTP",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Profiles:   make([]profileBundleProfile, 0, len(profiles)),
	}
	for _, profile := range profiles {
		bundle.Profiles = append(bundle.Profiles, exportedProfile(profile))
	}
	payload, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return 0, err
	}
	payload = append(payload, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return 0, err
	}
	temp, err := os.CreateTemp(dir, ".ghostftp-sites-*.tmp")
	if err != nil {
		return 0, err
	}
	tempPath := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}
	if err := temp.Chmod(0600); err != nil {
		cleanup()
		return 0, err
	}
	if _, err := temp.Write(payload); err != nil {
		cleanup()
		return 0, err
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return 0, err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return 0, err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return 0, err
	}
	return len(profiles), nil
}

func readProfileBundle(path string) (profileBundle, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return profileBundle{}, errors.New("import path is empty")
	}
	file, err := os.Open(path)
	if err != nil {
		return profileBundle{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxProfileBundleBytes+1))
	if err != nil {
		return profileBundle{}, err
	}
	if len(data) == 0 || len(data) > maxProfileBundleBytes {
		return profileBundle{}, errors.New("site bundle is empty or too large")
	}
	var bundle profileBundle
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bundle); err != nil {
		return profileBundle{}, fmt.Errorf("invalid site bundle: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return profileBundle{}, errors.New("invalid site bundle")
	}
	if bundle.Schema != profileBundleSchema || bundle.Product != "Ghost FTP" {
		return profileBundle{}, errors.New("unsupported Ghost FTP site bundle")
	}
	if len(bundle.Profiles) > maxImportedProfiles {
		return profileBundle{}, errors.New("site bundle contains too many profiles")
	}
	return bundle, nil
}

func sameImportedProfile(existing model.PublicProfile, incoming profileBundleProfile) bool {
	return strings.EqualFold(strings.TrimSpace(existing.Name), strings.TrimSpace(incoming.Name)) &&
		strings.EqualFold(strings.TrimSpace(existing.Protocol), strings.TrimSpace(incoming.Protocol)) &&
		strings.EqualFold(strings.TrimSpace(existing.Host), strings.TrimSpace(incoming.Host)) &&
		existing.Port == incoming.Port &&
		strings.EqualFold(strings.TrimSpace(existing.Username), strings.TrimSpace(incoming.Username)) &&
		strings.TrimSpace(existing.RemotePath) == strings.TrimSpace(incoming.RemotePath) &&
		strings.TrimSpace(existing.LocalPath) == strings.TrimSpace(incoming.LocalPath)
}

func (e *Engine) ImportProfiles(path string) (ProfileImportResult, error) {
	bundle, err := readProfileBundle(path)
	if err != nil {
		return ProfileImportResult{}, err
	}
	existing, err := e.profiles.List()
	if err != nil {
		return ProfileImportResult{}, err
	}
	result := ProfileImportResult{}
	for _, incoming := range bundle.Profiles {
		duplicate := false
		for _, profile := range existing {
			if sameImportedProfile(profile, incoming) {
				duplicate = true
				break
			}
		}
		if duplicate {
			result.Skipped++
			continue
		}
		saved, err := e.profiles.Save(model.ProfileInput{
			Name:           incoming.Name,
			Protocol:       incoming.Protocol,
			Host:           incoming.Host,
			Port:           incoming.Port,
			Username:       incoming.Username,
			PrivateKeyPath: incoming.PrivateKeyPath,
			RemotePath:     incoming.RemotePath,
			LocalPath:      incoming.LocalPath,
		})
		if err != nil {
			return result, fmt.Errorf("import %q: %w", incoming.Name, err)
		}
		existing = append(existing, saved)
		result.Imported++
	}
	return result, nil
}
