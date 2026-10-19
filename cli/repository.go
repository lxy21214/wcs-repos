package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Downloader struct {
	Client  *http.Client
	Retries int
	Pause   func(time.Duration)
}

func newDownloader() *Downloader {
	return &Downloader{Client: &http.Client{Timeout: 15 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("refusing non-HTTPS redirect")
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}}, Retries: 5, Pause: time.Sleep}
}

func (d *Downloader) transfer(address string, consume func(io.Reader) error) error {
	var last error
	for attempt := 0; attempt <= d.Retries; attempt++ {
		if attempt > 0 {
			d.Pause(time.Duration(attempt) * time.Second)
		}
		req, err := http.NewRequest("GET", address, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "wcs-repo/"+version)
		resp, err := d.Client.Do(req)
		if err != nil {
			last = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			last = fmt.Errorf("GET %s: HTTP %d", address, resp.StatusCode)
			if resp.StatusCode == 408 || resp.StatusCode == 429 || resp.StatusCode >= 500 {
				continue
			}
			return last
		}
		err = consume(resp.Body)
		resp.Body.Close()
		if err == nil {
			return nil
		}
		last = err
	}
	return last
}

func (d *Downloader) read(address string) ([]byte, error) {
	var data []byte
	err := d.transfer(address, func(r io.Reader) error {
		var err error
		data, err = io.ReadAll(io.LimitReader(r, 64<<20+1))
		if len(data) > 64<<20 {
			return fmt.Errorf("metadata exceeds 64 MiB")
		}
		return err
	})
	return data, err
}

func (d *Downloader) download(address, destination string) error {
	return d.transfer(address, func(r io.Reader) error {
		f, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, r)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}

type Repository struct {
	Root, Slug, Commit string
	Downloader         *Downloader
}

func openRepository(location, ref string, d *Downloader) (*Repository, error) {
	if location == "" {
		return nil, fmt.Errorf("set --repo or SOFTWARE_REPO_URL; there is no default repository")
	}
	r := &Repository{Downloader: d}
	if !strings.Contains(location, "://") {
		root, err := filepath.Abs(location)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(root)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("repository is not a directory: %s", root)
		}
		r.Root = root
		return r, nil
	}
	u, err := url.Parse(location)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("repository must be a local directory or https://github.com/owner/repo")
	}
	r.Slug = strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	parts := strings.Split(r.Slug, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected GitHub owner/repository")
	}
	for _, part := range parts {
		if err := segment(part); err != nil {
			return nil, err
		}
	}
	if ref == "" {
		ref = "main"
	}
	data, err := d.read("https://api.github.com/repos/" + escapePath(r.Slug) + "/commits/" + url.PathEscape(ref))
	if err != nil {
		return nil, err
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(data, &commit); err != nil {
		return nil, err
	}
	if len(commit.SHA) != 40 || strings.IndexFunc(commit.SHA, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) >= 0 {
		return nil, fmt.Errorf("GitHub returned invalid commit")
	}
	r.Commit = commit.SHA
	return r, nil
}

func escapePath(s string) string {
	parts := strings.Split(s, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func (r *Repository) address(rel string, lfs bool) string {
	base := "https://raw.githubusercontent.com/"
	if lfs {
		base = "https://media.githubusercontent.com/media/"
	}
	return base + escapePath(r.Slug) + "/" + r.Commit + "/" + escapePath(path.Clean(rel))
}

func (r *Repository) read(rel string) ([]byte, error) {
	if err := relative(rel); err != nil {
		return nil, err
	}
	if r.Root != "" {
		filename, err := localPath(r.Root, rel)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(filename)
	}
	return r.Downloader.read(r.address(rel, false))
}

func (r *Repository) load(software, version string) (*Index, string, error) {
	if err := segment(software); err != nil {
		return nil, "", err
	}
	if err := segment(version); err != nil {
		return nil, "", err
	}
	rel := software + "/" + version + "/index.json"
	data, err := r.read(rel)
	if err != nil {
		return nil, "", err
	}
	index, err := decodeIndex(data)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", rel, err)
	}
	if index.Software != software || index.Version != version {
		return nil, "", fmt.Errorf("index identity does not match path: %s", rel)
	}
	return index, rel, nil
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func lfsPointer(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer f.Close()
	b := make([]byte, 128)
	n, _ := f.Read(b)
	return bytes.HasPrefix(b[:n], []byte("version https://git-lfs.github.com/spec/v1\n"))
}

func (r *Repository) cacheFile(rel, destination string) error {
	if err := relative(rel); err != nil {
		return err
	}
	if r.Root != "" {
		filename, err := localPath(r.Root, rel)
		if err != nil {
			return err
		}
		if lfsPointer(filename) {
			return fmt.Errorf("%s is a Git LFS pointer; run git lfs pull in the local repository", rel)
		}
		return copyFile(filename, destination)
	}
	if err := r.Downloader.download(r.address(rel, false), destination); err != nil {
		return err
	}
	if lfsPointer(destination) {
		return r.Downloader.download(r.address(rel, true), destination)
	}
	return nil
}

func (r *Repository) fetch(p Package, indexPath, destination string) error {
	if info, err := os.Lstat(destination); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("destination must be a regular file: %s", destination)
		}
		return verifyFile(destination, p.SHA256)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".wcs-repo-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)
	if p.Cache != "" {
		err = r.cacheFile(path.Join(path.Dir(indexPath), p.Cache), name)
	} else {
		err = r.Downloader.download(p.URL, name)
	}
	if err != nil {
		return err
	}
	if err = verifyFile(name, p.SHA256); err != nil {
		return err
	}
	if err = os.Chmod(name, 0644); err != nil {
		return err
	}
	// Link commits the completed file without overwriting a concurrent writer.
	if err = os.Link(name, destination); os.IsExist(err) {
		return verifyFile(destination, p.SHA256)
	}
	return err
}
