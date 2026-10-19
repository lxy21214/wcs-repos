package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func checksum(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}
func testDownloader(handler transport) *Downloader {
	return &Downloader{Client: &http.Client{Transport: handler}, Retries: 2, Pause: func(time.Duration) {}}
}
func noNetwork(t *testing.T) *Downloader {
	t.Helper()
	return testDownloader(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected network: %s", r.URL)
		return nil, errors.New("network forbidden")
	})
}
func fixture(t *testing.T, schema int) (string, string) {
	t.Helper()
	root := t.TempDir()
	rel := "python/3.13t.16/index.json"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, rel)), 0755); err != nil {
		t.Fatal(err)
	}
	p := map[string]any{"version": "3.13.16", "filename": "python.tar.zst", "url": "https://upstream.invalid/python", "sha256": checksum("python"), "metadata": map[string]any{"unusual": true}}
	uv := map[string]any{"filename": "uv.whl", "url": "https://upstream.invalid/uv", "sha256": checksum("uv")}
	index := map[string]any{"schema_version": schema, "software": "python", "version": "3.13t.16", "metadata": map[string]any{"custom": "preserved"}}
	if schema == 4 {
		index["packages"] = map[string]any{"python": p, "uv": uv}
	} else {
		p["name"] = "python"
		uv["name"] = "uv"
		index["components"] = []any{p, uv}
	}
	if err := atomicJSON(filepath.Join(root, rel), index); err != nil {
		t.Fatal(err)
	}
	return root, rel
}
func invoke(t *testing.T, d *Downloader, args ...string) (string, error) {
	t.Helper()
	var out, log bytes.Buffer
	err := run(args, &out, &log, d)
	return out.String(), err
}

func TestFetchRequiresSelectionAndRepository(t *testing.T) {
	t.Setenv("SOFTWARE_REPO_URL", "")
	t.Setenv("SOFTWARE_REPO_REF", "")
	for _, args := range [][]string{{"list"}, {"fetch", "python", "3.13t.16", "--output", t.TempDir()}, {"fetch", "python", "3.13t.16", "--package", "python"}} {
		if _, err := invoke(t, noNetwork(t), args...); err == nil {
			t.Fatalf("accepted incomplete invocation: %v", args)
		}
	}
	if _, err := invoke(t, noNetwork(t), "--help"); err != nil {
		t.Fatal(err)
	}
}

func TestCacheAndFetchPreserveV4Metadata(t *testing.T) {
	for _, schema := range []int{4} {
		t.Run(fmt.Sprint(schema), func(t *testing.T) {
			root, rel := fixture(t, schema)
			calls := 0
			d := testDownloader(func(r *http.Request) (*http.Response, error) {
				calls++
				return response(200, strings.TrimPrefix(r.URL.Path, "/")), nil
			})
			_, err := invoke(t, d, "--repo", root, "cache", "python", "3.13t.16", "--package", "python", "--package", "uv")
			if err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatal(calls)
			}
			data, _ := os.ReadFile(filepath.Join(root, rel))
			index, err := decodeIndex(data)
			if err != nil {
				t.Fatal(err)
			}
			if index.Schema != schema || !bytes.Contains(index.Raw["metadata"], []byte("preserved")) {
				t.Fatal("metadata changed")
			}
			if index.Packages["python"].Cache != "./python.tar.zst" || !bytes.Contains(data, []byte("unusual")) {
				t.Fatal("cache or metadata missing")
			}
			out := t.TempDir()
			args := []string{"fetch", "python", "3.13t.16", "--repo", root, "--package", "uv", "--output", out, "--json"}
			result, err := invoke(t, noNetwork(t), args...)
			if err != nil {
				t.Fatal(err)
			}
			var found struct{ Packages map[string]string }
			if err = json.Unmarshal([]byte(result), &found); err != nil {
				t.Fatal(err)
			}
			if len(found.Packages) != 1 || found.Packages["uv"] != filepath.Join(out, "uv/uv.whl") {
				t.Fatal(result)
			}
			if _, err = os.Stat(filepath.Join(out, "python")); !os.IsNotExist(err) {
				t.Fatal("unselected package fetched")
			}
			if _, err = invoke(t, noNetwork(t), args...); err != nil {
				t.Fatal(err)
			}
			os.WriteFile(filepath.Join(out, "uv/uv.whl"), []byte("edited"), 0644)
			if _, err = invoke(t, noNetwork(t), args...); err == nil || !strings.Contains(err.Error(), "SHA-256") {
				t.Fatal(err)
			}
			if content, _ := os.ReadFile(filepath.Join(out, "uv/uv.whl")); string(content) != "edited" {
				t.Fatal("existing file overwritten")
			}
		})
	}
}

func TestListShowCheckIndexAndOpaqueVersions(t *testing.T) {
	root, _ := fixture(t, 4)
	// handoff is explicitly outside the catalogue and must not be inspected.
	os.MkdirAll(filepath.Join(root, "handoff/broken"), 0755)
	os.WriteFile(filepath.Join(root, "handoff/broken/index.json"), []byte("invalid"), 0644)
	out, err := invoke(t, noNetwork(t), "list", "python", "--repo", root)
	if err != nil || out != "3.13t.16\n" {
		t.Fatal(out, err)
	}
	out, err = invoke(t, noNetwork(t), "show", "python", "3.13t.16", "--repo", root)
	if err != nil || !strings.Contains(out, `"packages"`) {
		t.Fatal(out, err)
	}
	if _, err = invoke(t, noNetwork(t), "check", "--repo", root); err != nil {
		t.Fatal(err)
	}
	if _, err = invoke(t, noNetwork(t), "index", "--repo", root); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "manifest.json"))
	if !bytes.Contains(data, []byte("3.13t.16")) || bytes.Contains(data, []byte("handoff")) {
		t.Fatal(string(data))
	}
	data, _ = os.ReadFile(filepath.Join(root, "index-bundle.json"))
	if !bytes.Contains(data, []byte(`"_path": "python/3.13t.16/index.json"`)) {
		t.Fatal(string(data))
	}
}

func TestChecksumFailureAndBrokenCacheNeverFallback(t *testing.T) {
	root, rel := fixture(t, 4)
	out := t.TempDir()
	d := testDownloader(func(*http.Request) (*http.Response, error) { return response(200, "wrong content"), nil })
	args := []string{"fetch", "python", "3.13t.16", "--repo", root, "--package", "python", "--output", out}
	if _, err := invoke(t, d, args...); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatal(err)
	}
	files, _ := os.ReadDir(filepath.Join(out, "python"))
	if len(files) != 0 {
		t.Fatal("partial files retained")
	}
	data, _ := os.ReadFile(filepath.Join(root, rel))
	i, _ := decodeIndex(data)
	i.setCache("python", "./missing.tar.zst")
	atomicJSON(filepath.Join(root, rel), i.Raw)
	if _, err := invoke(t, noNetwork(t), args...); err == nil {
		t.Fatal("missing declared cache should fail")
	}
	if _, err := invoke(t, noNetwork(t), "check", "--repo", root); err == nil {
		t.Fatal("check accepted missing declared cache")
	}
}

func TestNetworkRetry(t *testing.T) {
	calls := 0
	d := testDownloader(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("temporary proxy failure")
		}
		if calls == 2 {
			return response(503, "retry"), nil
		}
		return response(200, "ok"), nil
	})
	data, err := d.read("https://example.invalid/test")
	if err != nil || string(data) != "ok" || calls != 3 {
		t.Fatal(string(data), calls, err)
	}
	calls = 0
	d.Client.Transport = transport(func(*http.Request) (*http.Response, error) { calls++; return response(404, "missing"), nil })
	if _, err = d.read("https://example.invalid/test"); err == nil || calls != 1 {
		t.Fatal(calls, err)
	}
}

func TestGitHubPinsCommitAndReadsLFS(t *testing.T) {
	root, rel := fixture(t, 4)
	data, _ := os.ReadFile(filepath.Join(root, rel))
	i, _ := decodeIndex(data)
	i.setCache("uv", "./uv.whl")
	data, _ = jsonBytes(i.Raw)
	commit := strings.Repeat("a", 40)
	calls := []string{}
	d := testDownloader(func(r *http.Request) (*http.Response, error) {
		address := r.URL.String()
		calls = append(calls, address)
		switch address {
		case "https://api.github.com/repos/owner/repo/commits/main":
			return response(200, `{"sha":"`+commit+`"}`), nil
		case "https://raw.githubusercontent.com/owner/repo/" + commit + "/" + rel:
			return response(200, string(data)), nil
		case "https://raw.githubusercontent.com/owner/repo/" + commit + "/python/3.13t.16/uv.whl":
			return response(200, "version https://git-lfs.github.com/spec/v1\noid sha256:example\nsize 2\n"), nil
		case "https://media.githubusercontent.com/media/owner/repo/" + commit + "/python/3.13t.16/uv.whl":
			return response(200, "uv"), nil
		default:
			t.Errorf("unexpected URL: %s", address)
			return response(404, ""), nil
		}
	})
	out, err := invoke(t, d, "--repo", "https://github.com/owner/repo", "--ref", "main", "fetch", "python", "3.13t.16", "--package", "uv", "--output", t.TempDir(), "--json")
	if err != nil || len(calls) != 4 || !strings.Contains(out, commit) {
		t.Fatal(out, calls, err)
	}
}

func TestPathsAndUnknownPackages(t *testing.T) {
	root, rel := fixture(t, 4)
	if _, err := invoke(t, noNetwork(t), "show", "../python", "3.13t.16", "--repo", root); err == nil {
		t.Fatal("accepted traversal")
	}
	output := filepath.Join(t.TempDir(), "not-created")
	if _, err := invoke(t, noNetwork(t), "fetch", "python", "3.13t.16", "--repo", root, "--package", "missing", "--output", output); err == nil {
		t.Fatal("accepted missing package")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("created output for invalid selection")
	}
	data, _ := os.ReadFile(filepath.Join(root, rel))
	i, _ := decodeIndex(data)
	i.setCache("python", "../../escape")
	data, _ = jsonBytes(i.Raw)
	if _, err := decodeIndex(data); err == nil {
		t.Fatal("accepted escaping cache path")
	}
	os.Symlink(t.TempDir(), filepath.Join(root, "outside"))
	if _, err := localPath(root, "outside/file"); err == nil {
		t.Fatal("followed escaping symlink")
	}
}

func TestOnlyV4AndProtocolFieldsAreAccepted(t *testing.T) {
	root, rel := fixture(t, 3)
	data, _ := os.ReadFile(filepath.Join(root, rel))
	if _, err := decodeIndex(data); err == nil {
		t.Fatal("v3 must not be accepted")
	}
	root, rel = fixture(t, 4)
	data, _ = os.ReadFile(filepath.Join(root, rel))
	for name, mutate := range map[string]func(map[string]any){
		"top-level field":  func(d map[string]any) { d["platform"] = "linux" },
		"metadata type":    func(d map[string]any) { d["metadata"] = []string{"wrong"} },
		"package field":    func(d map[string]any) { d["packages"].(map[string]any)["uv"].(map[string]any)["role"] = "tool" },
		"package metadata": func(d map[string]any) { d["packages"].(map[string]any)["uv"].(map[string]any)["metadata"] = nil },
		"empty cache":      func(d map[string]any) { d["packages"].(map[string]any)["uv"].(map[string]any)["cache"] = "" },
	} {
		t.Run(name, func(t *testing.T) {
			var document map[string]any
			json.Unmarshal(data, &document)
			mutate(document)
			invalid, _ := json.Marshal(document)
			if _, err := decodeIndex(invalid); err == nil {
				t.Fatal("accepted invalid v4 document")
			}
		})
	}
}

func TestCheckDetectsStaleSummaries(t *testing.T) {
	root, rel := fixture(t, 4)
	if _, err := invoke(t, noNetwork(t), "--repo", root, "index"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, rel))
	i, _ := decodeIndex(data)
	i.Raw["metadata"] = json.RawMessage(`{"updated":true}`)
	atomicJSON(filepath.Join(root, rel), i.Raw)
	if _, err := invoke(t, noNetwork(t), "--repo", root, "check"); err == nil || !strings.Contains(err.Error(), "outdated") {
		t.Fatal(err)
	}
	if _, err := invoke(t, noNetwork(t), "--repo", root, "index"); err != nil {
		t.Fatal(err)
	}
	if _, err := invoke(t, noNetwork(t), "--repo", root, "check"); err != nil {
		t.Fatal(err)
	}
}
