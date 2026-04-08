package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const releasesAPI = "https://api.github.com/repos/ashutoshsinghai/termshare/releases?per_page=10"

type ghRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Check for updates and upgrade termshare",
	RunE:  runUpgrade,
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	fmt.Println("Checking for updates...")

	releases, err := fetchReleases()
	if err != nil {
		return fmt.Errorf("failed to fetch releases: %w", err)
	}
	if len(releases) == 0 {
		fmt.Println("No releases found.")
		return nil
	}

	latest := releases[0].TagName
	current := Version

	fmt.Printf("Current version : %s\n", current)
	fmt.Printf("Latest version  : %s\n", latest)

	if current == latest {
		fmt.Println("\nAlready up to date.")
		return nil
	}

	// Build version list for dropdown
	items := make([]string, len(releases))
	for i, r := range releases {
		label := r.TagName
		if i == 0 {
			label += "  (latest)"
		}
		if r.TagName == current {
			label += "  (current)"
		}
		items[i] = label
	}

	prompt := promptui.Select{
		Label: "Select version to install",
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	selected := releases[idx].TagName
	return installVersion(selected)
}

func fetchReleases() ([]ghRelease, error) {
	resp, err := http.Get(releasesAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var releases []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func installVersion(version string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map to release asset naming
	archName := goarch
	if goarch == "amd64" {
		archName = "amd64"
	}

	var assetURL string
	if goos == "windows" {
		assetURL = fmt.Sprintf(
			"https://github.com/ashutoshsinghai/termshare/releases/download/%s/termshare_windows_%s.zip",
			version, archName,
		)
	} else {
		assetURL = fmt.Sprintf(
			"https://github.com/ashutoshsinghai/termshare/releases/download/%s/termshare_%s_%s.tar.gz",
			version, goos, archName,
		)
	}

	fmt.Printf("Downloading %s...\n", assetURL)

	resp, err := http.Get(assetURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Extract binary to temp file
	tmp, err := os.CreateTemp("", "termshare-upgrade-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if goos == "windows" {
		if err := extractFromZip(resp.Body, tmp); err != nil {
			return err
		}
	} else {
		if err := extractFromTarGz(resp.Body, tmp); err != nil {
			return err
		}
	}
	tmp.Close()

	// Replace current binary
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not find current binary: %w", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)

	if goos == "windows" {
		// Can't replace a running binary on Windows — place next to it
		newPath := strings.TrimSuffix(exe, ".exe") + ".new.exe"
		if err := copyFile(tmp.Name(), newPath); err != nil {
			return err
		}
		fmt.Printf("\nDownloaded to %s\n", newPath)
		fmt.Println("Rename it to termshare.exe to complete the upgrade.")
		return nil
	}

	if err := os.Chmod(tmp.Name(), 0755); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		if !os.IsPermission(err) {
			return fmt.Errorf("failed to replace binary: %w", err)
		}
		// Permission denied — retry with sudo
		fmt.Println("Permission denied. Retrying with sudo...")
		if sudoErr := sudoMove(tmp.Name(), exe); sudoErr != nil {
			return fmt.Errorf("sudo failed: %w", sudoErr)
		}
	}

	// Remove macOS quarantine
	if goos == "darwin" {
		_ = exec_xattr(exe)
	}

	fmt.Printf("\ntermshare upgraded to %s\n", version)
	return nil
}

func extractFromTarGz(r io.Reader, dst *os.File) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) == "termshare" {
			_, err = io.Copy(dst, tr)
			return err
		}
	}
	return fmt.Errorf("termshare binary not found in archive")
}

func extractFromZip(r io.Reader, dst *os.File) error {
	// zip requires io.ReaderAt — buffer to temp first
	tmp, err := os.CreateTemp("", "termshare-zip-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	size, err := io.Copy(tmp, r)
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(tmp, size)
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) == "termshare.exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			_, err = io.Copy(dst, rc)
			return err
		}
	}
	return fmt.Errorf("termshare.exe not found in zip")
}

func sudoMove(src, dst string) error {
	cmd := osexec.Command("sudo", "mv", src, dst)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
