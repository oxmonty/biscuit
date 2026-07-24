package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDetectInstall(t *testing.T) {
	// given/then: the executable path names the install channel
	cases := map[string]installChannel{
		"/opt/homebrew/Caskroom/biscuit-cli/0.1.0/biscuit":      installBrew,
		"/opt/homebrew/Caskroom/biscuit-cli@next/0.1.0/biscuit": installBrew,
		"/usr/local/Cellar/biscuit/0.1.0/bin/biscuit":           installBrew,
		"/usr/lib/node_modules/@oxmonty/biscuit-darwin-arm64/bin/biscuit": installNPM,
		"/Users/x/.biscuit/bin/biscuit":                         installBare,
		"/usr/local/bin/biscuit":                                installBare,
	}
	for path, want := range cases {
		if got := detectInstall(path); got != want {
			t.Errorf("detectInstall(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestReleaseChannel(t *testing.T) {
	// given/then: brew reads the cask path; others read the version string
	cases := []struct {
		install installChannel
		path    string
		current string
		want    string
	}{
		{installBrew, "/opt/homebrew/Caskroom/biscuit-cli@next/x/biscuit", "0.1.0-alpha.5", "next"},
		{installBrew, "/opt/homebrew/Caskroom/biscuit-cli/x/biscuit", "0.1.0", "stable"},
		{installBare, "/x/biscuit", "0.1.0-alpha.5", "next"},
		{installBare, "/x/biscuit", "1.2.0", "stable"},
		{installBare, "/x/biscuit", "dev", "next"},
		{installNPM, "/x/node_modules/y/biscuit", "0.1.0-rc.1", "next"},
	}
	for _, c := range cases {
		if got := releaseChannel(c.install, c.path, c.current); got != c.want {
			t.Errorf("releaseChannel(%v, %q, %q) = %q, want %q", c.install, c.path, c.current, got, c.want)
		}
	}
}

func TestVerifyChecksum(t *testing.T) {
	// given: an archive and a checksums.txt in goreleaser's format
	archive := []byte("archive-bytes")
	sum := sha256.Sum256(archive)
	sums := []byte(fmt.Sprintf("%s  biscuit_1.0.0_darwin_arm64.tar.gz\nother  other.tar.gz\n", hex.EncodeToString(sum[:])))

	// then: the matching entry verifies; a tampered archive fails; a missing
	// entry fails
	if err := verifyChecksum(archive, sums, "biscuit_1.0.0_darwin_arm64.tar.gz"); err != nil {
		t.Errorf("valid checksum rejected: %v", err)
	}
	if err := verifyChecksum([]byte("tampered"), sums, "biscuit_1.0.0_darwin_arm64.tar.gz"); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("tampered archive accepted: %v", err)
	}
	if err := verifyChecksum(archive, sums, "missing.tar.gz"); err == nil {
		t.Error("missing entry accepted")
	}
}

// fakeRelease builds a tar.gz holding a fake biscuit binary plus its
// checksums.txt line.
func fakeRelease(t *testing.T, contents string) (archive, sums []byte, filename string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "biscuit", Mode: 0o755, Size: int64(len(contents)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(tw, contents); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()
	archive = buf.Bytes()
	sum := sha256.Sum256(archive)
	filename = fmt.Sprintf("biscuit_2.0.0_darwin_%s.tar.gz", runtime.GOARCH)
	sums = []byte(hex.EncodeToString(sum[:]) + "  " + filename + "\n")
	return archive, sums, filename
}

func runUpgradeWith(t *testing.T, opts *upgradeOptions, stdin string) (string, string, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(stdin))
	opts.fillDefaults()
	err := runUpgrade(cmd, opts)
	return out.String(), errOut.String(), err
}

func TestBareUpgradeSelfSwapsWithChecksum(t *testing.T) {
	// given: a fake GitHub serving a newer next release and a bare install
	archive, sums, filename := fakeRelease(t, "#!/bin/sh\necho new\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/oxmonty/biscuit/releases":
			_, _ = io.WriteString(w, `[{"tag_name":"v2.0.0-alpha.1"}]`)
		case strings.HasSuffix(r.URL.Path, filename):
			_, _ = w.Write(archive)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			_, _ = w.Write(sums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "biscuit")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	// when: upgrading on the next channel — the fake release version must be
	// what selfSwap downloads, so pin it
	opts := &upgradeOptions{
		version:        "2.0.0",
		exePath:        func() (string, error) { return exe, nil },
		apiBase:        srv.URL,
		dlBase:         srv.URL,
		currentVersion: "1.0.0-alpha.1",
		goos:           "darwin",
	}
	out, _, err := runUpgradeWith(t, opts, "")
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// then: the binary is swapped and executable
	got, err := os.ReadFile(exe)
	if err != nil || !strings.Contains(string(got), "echo new") {
		t.Errorf("binary not swapped: %q, %v", got, err)
	}
	info, _ := os.Stat(exe)
	if info.Mode()&0o111 == 0 {
		t.Error("swapped binary not executable")
	}
	if !strings.Contains(out, "upgraded biscuit") {
		t.Errorf("no success message: %q", out)
	}
}

func TestBareUpgradeRejectsTamperedArchive(t *testing.T) {
	// given: an archive that doesn't match its checksums.txt
	archive, _, filename := fakeRelease(t, "payload")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, filename):
			_, _ = w.Write(archive)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			_, _ = io.WriteString(w, strings.Repeat("0", 64)+"  "+filename+"\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "biscuit")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	// when/then: the upgrade fails on checksum and the binary is untouched
	opts := &upgradeOptions{
		version: "2.0.0",
		exePath: func() (string, error) { return exe, nil },
		apiBase: srv.URL, dlBase: srv.URL,
		currentVersion: "1.0.0", goos: "darwin", yes: true,
	}
	_, _, err := runUpgradeWith(t, opts, "")
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want checksum mismatch", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("binary replaced despite bad checksum: %q", got)
	}
}

func TestStableChannelWithNoStableRelease(t *testing.T) {
	// given: /releases/latest 404s (no stable release yet)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	// when/then: a clear error points at --channel next
	opts := &upgradeOptions{
		channel: "stable",
		exePath: func() (string, error) { return "/x/.biscuit/bin/biscuit", nil },
		apiBase: srv.URL, dlBase: srv.URL,
		currentVersion: "1.0.0", goos: "darwin",
	}
	_, _, err := runUpgradeWith(t, opts, "")
	if err == nil || !strings.Contains(err.Error(), "no stable release") {
		t.Fatalf("err = %v, want no-stable-release guidance", err)
	}
}

func TestBrewUpgradeExecsBrew(t *testing.T) {
	// given: a brew next-cask install with a recorded exec seam
	var calls [][]string
	opts := &upgradeOptions{
		exePath: func() (string, error) {
			return "/opt/homebrew/Caskroom/biscuit-cli@next/0.1.0/biscuit", nil
		},
		execCommand: func(_, _ io.Writer, name string, args ...string) error {
			calls = append(calls, append([]string{name}, args...))
			return nil
		},
		currentVersion: "0.1.0-alpha.5", goos: "darwin",
	}

	// when: upgrading without switching channels
	if _, _, err := runUpgradeWith(t, opts, ""); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// then: brew's own upgrade runs against the next cask
	want := []string{"brew", "upgrade", "--cask", "biscuit-cli@next"}
	if len(calls) != 1 || strings.Join(calls[0], " ") != strings.Join(want, " ") {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestBrewChannelSwitchConfirmsAndReinstalls(t *testing.T) {
	// given: a stable brew install asked to cross to next, user confirms
	var calls [][]string
	opts := &upgradeOptions{
		channel: "next",
		exePath: func() (string, error) {
			return "/opt/homebrew/Caskroom/biscuit-cli/0.1.0/biscuit", nil
		},
		execCommand: func(_, _ io.Writer, name string, args ...string) error {
			calls = append(calls, append([]string{name}, args...))
			return nil
		},
		currentVersion: "0.1.0", goos: "darwin",
	}

	// when: confirming with y
	if _, _, err := runUpgradeWith(t, opts, "y\n"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// then: uninstall old cask, install the fully-qualified new one
	if len(calls) != 2 ||
		strings.Join(calls[0], " ") != "brew uninstall --cask biscuit-cli" ||
		strings.Join(calls[1], " ") != "brew install --cask oxmonty/tap/biscuit-cli@next" {
		t.Errorf("calls = %v", calls)
	}
}

func TestNPMUpgradeExecsNPM(t *testing.T) {
	// given: an npm install on the next dist-tag
	var calls [][]string
	opts := &upgradeOptions{
		exePath: func() (string, error) {
			return "/usr/lib/node_modules/@oxmonty/biscuit-darwin-arm64/bin/biscuit", nil
		},
		execCommand: func(_, _ io.Writer, name string, args ...string) error {
			calls = append(calls, append([]string{name}, args...))
			return nil
		},
		currentVersion: "0.1.0-alpha.5", goos: "darwin",
	}

	// when
	if _, _, err := runUpgradeWith(t, opts, ""); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// then: npm reinstalls from its own dist-tag
	if len(calls) != 1 || strings.Join(calls[0], " ") != "npm install -g biscuit-cli@next" {
		t.Errorf("calls = %v", calls)
	}
}

func TestVersionPinOnBrewConfirms(t *testing.T) {
	// given: a brew install pinned to an exact version, user declines
	opts := &upgradeOptions{
		version: "0.1.0-alpha.4",
		exePath: func() (string, error) {
			return "/opt/homebrew/Caskroom/biscuit-cli@next/0.1.0/biscuit", nil
		},
		currentVersion: "0.1.0-alpha.5", goos: "darwin",
	}

	// when/then: declining aborts before any download
	_, errOut, err := runUpgradeWith(t, opts, "n\n")
	if err == nil || err.Error() != "aborted" {
		t.Fatalf("err = %v, want aborted", err)
	}
	if !strings.Contains(errOut, "takes ownership") {
		t.Errorf("prompt missing ownership warning: %q", errOut)
	}
}
