package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Package struct {
	Version  string `json:"version,omitempty"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
	Cache    string `json:"cache,omitempty"`
}

type Index struct {
	Schema            int
	Software, Version string
	Packages          map[string]Package
	Raw               map[string]json.RawMessage
}

func segment(s string) error {
	if s == "" || s == "." || s == ".." || strings.ContainsAny(s, "/\\") || strings.IndexFunc(s, func(r rune) bool { return r < 32 || r == 127 }) >= 0 {
		return fmt.Errorf("invalid path segment %q", s)
	}
	return nil
}

func relative(s string) error {
	if s == "" || filepath.IsAbs(s) || strings.Contains(s, "\\") {
		return fmt.Errorf("invalid relative path %q", s)
	}
	for _, part := range strings.Split(strings.TrimPrefix(s, "./"), "/") {
		if err := segment(part); err != nil {
			return err
		}
	}
	return nil
}

func decodeIndex(data []byte) (*Index, error) {
	i := &Index{Packages: make(map[string]Package)}
	if err := json.Unmarshal(data, &i.Raw); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(i.Raw["schema_version"], &i.Schema); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(i.Raw["software"], &i.Software); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(i.Raw["version"], &i.Version); err != nil {
		return nil, err
	}
	if err := segment(i.Software); err != nil {
		return nil, err
	}
	if err := segment(i.Version); err != nil {
		return nil, err
	}
	if i.Schema != 4 {
		return nil, fmt.Errorf("unsupported schema_version: %d; expected 4", i.Schema)
	}
	if err := fields(i.Raw, "schema_version", "software", "version", "packages", "metadata"); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(i.Raw["packages"], &i.Packages); err != nil {
		return nil, err
	}
	var rawPackages map[string]json.RawMessage
	if err := json.Unmarshal(i.Raw["packages"], &rawPackages); err != nil {
		return nil, err
	}
	for name, raw := range rawPackages {
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, err
		}
		if err := fields(entry, "version", "filename", "url", "sha256", "cache", "metadata"); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		for _, key := range []string{"version", "cache"} {
			if value, ok := entry[key]; ok && (len(value) == 0 || value[0] != '"') {
				return nil, fmt.Errorf("%s.%s must be a string", name, key)
			}
		}
		if _, ok := entry["cache"]; ok {
			if err := relative(i.Packages[name].Cache); err != nil {
				return nil, err
			}
		}
	}
	if len(i.Packages) == 0 {
		return nil, fmt.Errorf("index has no packages")
	}
	for name, p := range i.Packages {
		if err := segment(name); err != nil {
			return nil, err
		}
		if err := segment(p.Filename); err != nil {
			return nil, err
		}
		u, err := url.Parse(p.URL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
			return nil, fmt.Errorf("%s: url must be HTTPS", name)
		}
		digest, err := hex.DecodeString(p.SHA256)
		if err != nil || len(digest) != sha256.Size {
			return nil, fmt.Errorf("%s: invalid sha256", name)
		}
		if p.Cache != "" {
			if err := relative(p.Cache); err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	return i, nil
}

func (i *Index) setCache(name, cache string) error {
	var packages map[string]map[string]json.RawMessage
	if err := json.Unmarshal(i.Raw["packages"], &packages); err != nil {
		return err
	}
	entry, ok := packages[name]
	if !ok {
		return fmt.Errorf("unknown package: %s", name)
	}
	entry["cache"], _ = json.Marshal(cache)
	i.Raw["packages"], _ = json.Marshal(packages)
	p := i.Packages[name]
	p.Cache = cache
	i.Packages[name] = p
	return nil
}

func fields(raw map[string]json.RawMessage, allowed ...string) error {
	names := make(map[string]bool)
	for _, key := range allowed {
		names[key] = true
	}
	for key := range raw {
		if !names[key] {
			return fmt.Errorf("unknown field %q; put software-specific fields in metadata", key)
		}
	}
	if value, ok := raw["metadata"]; ok {
		var metadata map[string]json.RawMessage
		if err := json.Unmarshal(value, &metadata); err != nil || metadata == nil {
			return fmt.Errorf("metadata must be an object")
		}
	}
	return nil
}

func digestFile(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifyFile(filename, expected string) error {
	actual, err := digestFile(filename)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("SHA-256 mismatch: %s (expected %s, got %s)", filename, expected, actual)
	}
	return nil
}

func jsonBytes(value any) ([]byte, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	return append(b, '\n'), err
}

func atomicJSON(filename string, value any) error {
	data, err := jsonBytes(value)
	if err != nil {
		return err
	}
	if old, err := os.ReadFile(filename); err == nil && bytes.Equal(old, data) {
		return nil
	}
	f, err := os.CreateTemp(filepath.Dir(filename), ".wcs-repo-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, filename)
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Resolve existing symlinks before allowing a repository path to be read or written.
func localPath(root, rel string) (string, error) {
	if err := relative(rel); err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return "", err
	}
	target := filepath.Join(canonical, filepath.FromSlash(rel))
	probe := target
	for {
		_, err = os.Lstat(probe)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		probe = filepath.Dir(probe)
	}
	resolved, err := filepath.EvalSymlinks(probe)
	if err != nil {
		return "", err
	}
	distance, err := filepath.Rel(canonical, resolved)
	if err != nil || distance == ".." || strings.HasPrefix(distance, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository: %s", rel)
	}
	return target, nil
}
