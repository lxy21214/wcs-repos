package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

const version = "0.2.0"
const usage = `wcs-repo: software indexes, downloads and verified caches

Usage:
  wcs-repo list [software]
  wcs-repo show <software> <version>
  wcs-repo fetch <software> <version> --package <name> [--package <name>] --output <directory>
  wcs-repo cache <software> <version> --package <name> [--package <name>]
  wcs-repo check
  wcs-repo index

Options (before or after the command):
  --repo <directory|https://github.com/owner/repo>  or SOFTWARE_REPO_URL; required
  --ref <branch|tag|commit>                       or SOFTWARE_REPO_REF; defaults to main
  --package <name>                               repeat to explicitly select packages
  --output <directory>                           required for fetch
  --json                                        machine-readable output
  --help, -h                                    show this help
  --version                                     show CLI version

cache, check and index operate on a local repository. No command installs software.
`

type options struct {
	repo, ref, output   string
	packages, args      []string
	json, help, version bool
}

func parseOptions(args []string) (options, error) {
	o := options{repo: os.Getenv("SOFTWARE_REPO_URL"), ref: os.Getenv("SOFTWARE_REPO_REF")}
	for n := 0; n < len(args); n++ {
		arg := args[n]
		switch arg {
		case "--help", "-h":
			o.help = true
		case "--version":
			o.version = true
		case "--json":
			o.json = true
		case "--repo", "--ref", "--output", "--package":
			if n+1 == len(args) || args[n+1] == "" {
				return o, fmt.Errorf("%s requires a value", arg)
			}
			n++
			switch arg {
			case "--repo":
				o.repo = args[n]
			case "--ref":
				o.ref = args[n]
			case "--output":
				o.output = args[n]
			case "--package":
				o.packages = append(o.packages, args[n])
			}
		default:
			if strings.HasPrefix(arg, "-") {
				return o, fmt.Errorf("unknown option: %s", arg)
			}
			o.args = append(o.args, arg)
		}
	}
	return o, nil
}

func scan(r *Repository) (map[string]*Index, error) {
	result := make(map[string]*Index)
	softwareDirs, err := os.ReadDir(r.Root)
	if err != nil {
		return nil, err
	}
	for _, software := range softwareDirs {
		if !software.IsDir() || software.Name() == "handoff" || strings.HasPrefix(software.Name(), ".") || strings.HasPrefix(software.Name(), "_") {
			continue
		}
		versions, err := os.ReadDir(filepath.Join(r.Root, software.Name()))
		if err != nil {
			return nil, err
		}
		for _, v := range versions {
			if !v.IsDir() || strings.HasPrefix(v.Name(), ".") || strings.HasPrefix(v.Name(), "_") {
				continue
			}
			rel := path.Join(software.Name(), v.Name(), "index.json")
			if _, err := os.Lstat(filepath.Join(r.Root, filepath.FromSlash(rel))); os.IsNotExist(err) {
				continue
			} else if err != nil {
				return nil, err
			}
			i, _, err := r.load(software.Name(), v.Name())
			if err != nil {
				return nil, err
			}
			result[rel] = i
		}
	}
	return result, nil
}

func catalogue(r *Repository) (map[string][]string, error) {
	result := make(map[string][]string)
	if r.Root != "" {
		indexes, err := scan(r)
		if err != nil {
			return nil, err
		}
		for _, rel := range sortedKeys(indexes) {
			i := indexes[rel]
			result[i.Software] = append(result[i.Software], i.Version)
		}
		return result, nil
	}
	data, err := r.read("manifest.json")
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Schema   int `json:"schema_version"`
		Software map[string]struct {
			Versions []string `json:"versions"`
		} `json:"software"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.Schema != 4 {
		return nil, fmt.Errorf("manifest must use schema_version 4")
	}
	if manifest.Software == nil {
		return nil, fmt.Errorf("manifest has no software entries")
	}
	for software, entry := range manifest.Software {
		if err := segment(software); err != nil {
			return nil, err
		}
		for _, v := range entry.Versions {
			if err := segment(v); err != nil {
				return nil, err
			}
		}
		result[software] = entry.Versions
	}
	return result, nil
}

func rebuild(r *Repository, indexes map[string]*Index) error {
	manifest := map[string]any{"schema_version": 4}
	manifestPath := filepath.Join(r.Root, "manifest.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	old, _ := manifest["software"].(map[string]any)
	software := make(map[string]any)
	versions := make(map[string][]string)
	entries := make([]map[string]json.RawMessage, 0, len(indexes))
	for _, rel := range sortedKeys(indexes) {
		i := indexes[rel]
		versions[i.Software] = append(versions[i.Software], i.Version)
		entry := make(map[string]json.RawMessage)
		for k, v := range i.Raw {
			entry[k] = v
		}
		entry["_path"], _ = json.Marshal(rel)
		entries = append(entries, entry)
	}
	for name, values := range versions {
		entry, _ := old[name].(map[string]any)
		if entry == nil {
			entry = make(map[string]any)
		}
		entry["versions"] = values
		entry["count"] = len(values)
		software[name] = entry
	}
	manifest["schema_version"] = 4
	manifest["software"] = software
	bundle := map[string]any{"schema_version": 4}
	bundlePath := filepath.Join(r.Root, "index-bundle.json")
	if data, err := os.ReadFile(bundlePath); err == nil {
		if err := json.Unmarshal(data, &bundle); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	bundle["schema_version"] = 4
	bundle["indexes"] = entries
	bundle["generated"] = time.Now().UTC().Format("2006-01-02")
	if err := atomicJSON(manifestPath, manifest); err != nil {
		return err
	}
	return atomicJSON(bundlePath, bundle)
}

func check(r *Repository, indexes map[string]*Index) (int, error) {
	if err := checkSummaries(r, indexes); err != nil {
		return 0, err
	}
	verified := 0
	for _, rel := range sortedKeys(indexes) {
		i := indexes[rel]
		for _, name := range sortedKeys(i.Packages) {
			p := i.Packages[name]
			cache := p.Cache
			if cache == "" {
				continue
			}
			filename, err := localPath(r.Root, path.Join(path.Dir(rel), cache))
			if err != nil {
				return 0, err
			}
			if lfsPointer(filename) {
				return 0, fmt.Errorf("%s is a Git LFS pointer; run git lfs pull", filename)
			}
			if err := verifyFile(filename, p.SHA256); err != nil {
				return 0, err
			}
			verified++
		}
	}
	return verified, nil
}

func checkSummaries(r *Repository, indexes map[string]*Index) error {
	data, err := os.ReadFile(filepath.Join(r.Root, "manifest.json"))
	if err == nil {
		var manifest struct {
			Schema   int `json:"schema_version"`
			Software map[string]struct {
				Versions []string `json:"versions"`
				Count    int      `json:"count"`
			} `json:"software"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return err
		}
		expected := make(map[string][]string)
		for _, i := range indexes {
			expected[i.Software] = append(expected[i.Software], i.Version)
		}
		if manifest.Schema != 4 || len(manifest.Software) != len(expected) {
			return fmt.Errorf("manifest is outdated; run wcs-repo index")
		}
		for name, versions := range expected {
			actual := manifest.Software[name]
			sort.Strings(versions)
			sort.Strings(actual.Versions)
			if actual.Count != len(versions) || !reflect.DeepEqual(actual.Versions, versions) {
				return fmt.Errorf("manifest entry %s is outdated; run wcs-repo index", name)
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err = os.ReadFile(filepath.Join(r.Root, "index-bundle.json"))
	if err == nil {
		var bundle struct {
			Schema  int                          `json:"schema_version"`
			Indexes []map[string]json.RawMessage `json:"indexes"`
		}
		if err := json.Unmarshal(data, &bundle); err != nil {
			return err
		}
		if bundle.Schema != 4 || len(bundle.Indexes) != len(indexes) {
			return fmt.Errorf("index-bundle is outdated; run wcs-repo index")
		}
		seen := make(map[string]bool)
		for _, entry := range bundle.Indexes {
			var rel string
			if err := json.Unmarshal(entry["_path"], &rel); err != nil {
				return err
			}
			i, ok := indexes[rel]
			if !ok || seen[rel] {
				return fmt.Errorf("invalid or duplicate bundle path: %s", rel)
			}
			seen[rel] = true
			delete(entry, "_path")
			var actual, expected any
			a, _ := json.Marshal(entry)
			b, _ := json.Marshal(i.Raw)
			json.Unmarshal(a, &actual)
			json.Unmarshal(b, &expected)
			if !reflect.DeepEqual(actual, expected) {
				return fmt.Errorf("bundle entry %s is outdated; run wcs-repo index", rel)
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func run(args []string, stdout, stderr io.Writer, d *Downloader) error {
	o, err := parseOptions(args)
	if err != nil {
		return err
	}
	if o.help {
		_, err = fmt.Fprint(stdout, usage)
		return err
	}
	if o.version {
		_, err = fmt.Fprintln(stdout, version)
		return err
	}
	if len(o.args) == 0 {
		return fmt.Errorf("command required; use --help")
	}
	command, positional := o.args[0], o.args[1:]
	switch command {
	case "list":
		if len(positional) > 1 {
			return fmt.Errorf("list accepts at most one software name")
		}
	case "show", "fetch", "cache":
		if len(positional) != 2 {
			return fmt.Errorf("%s requires software and version", command)
		}
	case "check", "index":
		if len(positional) != 0 {
			return fmt.Errorf("%s accepts no positional arguments", command)
		}
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
	if command == "fetch" || command == "cache" {
		if len(o.packages) == 0 {
			return fmt.Errorf("%s requires explicit --package selection; implicit bulk downloads are disabled", command)
		}
		if command == "fetch" && o.output == "" {
			return fmt.Errorf("fetch requires --output")
		}
	} else if len(o.packages) > 0 {
		return fmt.Errorf("--package is only supported by fetch and cache")
	}
	if command != "fetch" && o.output != "" {
		return fmt.Errorf("--output is only supported by fetch")
	}
	if (command == "cache" || command == "check" || command == "index") && strings.Contains(o.repo, "://") {
		return fmt.Errorf("%s requires a local repository", command)
	}
	r, err := openRepository(o.repo, o.ref, d)
	if err != nil {
		return err
	}
	printJSON := func(value any) error {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	if command == "list" {
		all, err := catalogue(r)
		if err != nil {
			return err
		}
		values := sortedKeys(all)
		if len(positional) == 1 {
			var found bool
			values, found = all[positional[0]]
			if !found {
				return fmt.Errorf("unknown software: %s", positional[0])
			}
		}
		if o.json {
			return printJSON(values)
		}
		for _, value := range values {
			fmt.Fprintln(stdout, value)
		}
		return nil
	}
	if command == "check" || command == "index" {
		indexes, err := scan(r)
		if err != nil {
			return err
		}
		if len(indexes) == 0 {
			return fmt.Errorf("no version indexes found")
		}
		result := map[string]any{"indexes": len(indexes)}
		if command == "check" {
			n, err := check(r, indexes)
			if err != nil {
				return err
			}
			result["verified_caches"] = n
		} else if err := rebuild(r, indexes); err != nil {
			return err
		}
		if o.json {
			return printJSON(result)
		}
		fmt.Fprintf(stdout, "%s: %d indexes\n", command, len(indexes))
		return nil
	}
	i, rel, err := r.load(positional[0], positional[1])
	if err != nil {
		return err
	}
	if command == "show" {
		return printJSON(i.Raw)
	}
	selected := make(map[string]Package)
	for _, name := range o.packages {
		p, ok := i.Packages[name]
		if !ok {
			return fmt.Errorf("package %q is not in %s", name, rel)
		}
		selected[name] = p
	}
	output := o.output
	if command == "fetch" {
		output, err = filepath.Abs(output)
		if err != nil {
			return err
		}
		if err = os.MkdirAll(output, 0755); err != nil {
			return err
		}
	} else {
		filenames := make(map[string]string)
		for _, p := range i.Packages {
			if previous, ok := filenames[p.Filename]; ok && !strings.EqualFold(previous, p.SHA256) {
				return fmt.Errorf("cache filename collision: %s", p.Filename)
			}
			filenames[p.Filename] = p.SHA256
		}
	}
	paths := make(map[string]string)
	for _, name := range sortedKeys(selected) {
		p := selected[name]
		var destination string
		if command == "fetch" {
			destination, err = localPath(output, name+"/"+p.Filename)
		} else {
			destination, err = localPath(r.Root, path.Join(path.Dir(rel), p.Filename))
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(stderr, "%s %s/%s: %s\n", command, i.Software, i.Version, name)
		if err = r.fetch(p, rel, destination); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		paths[name] = destination
		if command == "cache" {
			if err = i.setCache(name, "./"+p.Filename); err != nil {
				return err
			}
		}
	}
	if command == "cache" {
		filename, err := localPath(r.Root, rel)
		if err != nil {
			return err
		}
		if err := atomicJSON(filename, i.Raw); err != nil {
			return err
		}
	}
	if o.json {
		return printJSON(struct {
			Ref      string            `json:"ref,omitempty"`
			Packages map[string]string `json:"packages"`
		}{r.Commit, paths})
	}
	for _, name := range sortedKeys(paths) {
		fmt.Fprintf(stdout, "%s\t%s\n", name, paths[name])
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr, newDownloader()); err != nil {
		fmt.Fprintln(os.Stderr, "wcs-repo:", err)
		os.Exit(1)
	}
}
