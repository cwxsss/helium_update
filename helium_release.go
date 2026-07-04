package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	jsoniter "github.com/json-iterator/go"
)

const (
	heliumWindowsReleaseAPI = "https://api.github.com/repos/imputnet/helium-windows/releases/latest"
	heliumVersionMarker     = ".helium_updater_version"
	heliumPackageTypeZip    = "ZIP portable"
	heliumPackageTypeExe    = "EXE installer"
)

func getLatestHeliumInfo(sd *SettingsData) (ChromeInfo, error) {
	client, reqUrl := setProxy(sd, heliumWindowsReleaseAPI)
	logger.Infof("request Helium release API: %s", reqUrl)
	response, err := client.Get(reqUrl)
	if err != nil {
		logger.Errorln(err)
		return ChromeInfo{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ChromeInfo{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Errorln(err)
		return ChromeInfo{}, err
	}
	var release GithubRelease
	if err = jsoniter.Unmarshal(data, &release); err != nil {
		logger.Errorln(err)
		return ChromeInfo{}, err
	}
	assetIndex := selectHeliumAsset(release, getString(sd.heliumPackageType))
	if assetIndex >= 0 {
		asset := release.Assets[assetIndex]
		logger.Infof("selected Helium asset: name=%s, size=%d, url=%s", asset.Name, asset.Size, asset.BrowserDownloadURL)
		return ChromeInfo{
			Version: release.TagName,
			Size:    int64(asset.Size),
			Urls:    []string{asset.BrowserDownloadURL},
		}, nil
	}
	return ChromeInfo{}, fmt.Errorf("no Helium Windows x64 asset found in release %s", release.TagName)
}

func selectHeliumAsset(release GithubRelease, packageType string) int {
	if packageType == heliumPackageTypeExe {
		if i := findHeliumAsset(release, "_x64-installer.exe", true); i >= 0 {
			return i
		}
		if i := findHeliumAsset(release, "_x64-windows.zip", false); i >= 0 {
			return i
		}
	} else {
		if i := findHeliumAsset(release, "_x64-windows.zip", false); i >= 0 {
			return i
		}
		if i := findHeliumAsset(release, "_x64-installer.exe", true); i >= 0 {
			return i
		}
	}
	return findHeliumAsset(release, "_x64", true)
}

func findHeliumAsset(release GithubRelease, pattern string, excludeMini bool) int {
	for i, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, pattern) && (!excludeMini || !strings.Contains(name, "mini")) {
			return i
		}
	}
	return -1
}

func installHeliumPackage(packagePath, targetDir string) error {
	logger.Infof("install package: packagePath=%s, targetDir=%s", packagePath, targetDir)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return err
	}
	removeStaleHeliumPackages(targetDir)
	tempDir, err := os.MkdirTemp(filepath.Dir(targetDir), ".helium_extract_")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err = extractArchiveWith7Zip(packagePath, tempDir); err != nil {
		return err
	}
	if err = extractNestedHeliumArchive(tempDir); err != nil {
		return err
	}
	logger.Infof("extracted package to tempDir=%s", tempDir)
	appDir, err := findHeliumApplicationDir(tempDir)
	if err != nil {
		return err
	}
	logger.Infof("found Helium application dir: %s", appDir)
	if err = copyDirContents(appDir, targetDir); err != nil {
		return err
	}
	removeStaleHeliumPackages(targetDir)
	return nil
}

func extractNestedHeliumArchive(root string) error {
	var nestedArchive string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(d.Name(), "helium.7z") {
			nestedArchive = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil || nestedArchive == "" {
		return err
	}
	nestedTarget := filepath.Join(root, ".helium_nested")
	logger.Infof("extract nested Helium archive: %s", nestedArchive)
	return extractArchiveWith7Zip(nestedArchive, nestedTarget)
}

func findHeliumApplicationDir(root string) (string, error) {
	var fallback string
	var preferred string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !isHeliumExecutableName(d.Name()) {
			return nil
		}
		dir := filepath.Dir(path)
		logger.Infof("found Helium executable candidate: %s", path)
		if fallback == "" {
			fallback = dir
		}
		if strings.EqualFold(filepath.Base(dir), "Application") {
			preferred = dir
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if preferred != "" {
		return preferred, nil
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("Helium executable not found in extracted package")
}

func copyDirContents(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil || rel == "." {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, os.ModePerm)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}
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
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func removeStaleHeliumPackages(dir string) {
	patterns := []string{
		filepath.Join(dir, "helium_*installer.exe"),
		filepath.Join(dir, "helium_*windows.zip"),
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			logger.Infof("remove stale Helium package: %s", match)
			_ = os.Remove(match)
		}
	}
}

func extractArchiveWith7Zip(filePath, targetDir string) error {
	configPath := getConfigPath()
	zipDir := filepath.Dir(configPath)
	if !fileExist(zipDir) {
		_ = os.MkdirAll(zipDir, os.ModePerm)
	}
	zipExePath := filepath.Join(zipDir, "7za.exe")
	if !fileExist(zipExePath) {
		var data []byte
		if runtime.GOARCH == "386" {
			data = resourceAssets7z7za386Exe.Content()
		} else if runtime.GOARCH == "amd64" {
			data = resourceAssets7z7zaAmd64Exe.Content()
		} else if runtime.GOARCH == "arm64" {
			data = resourceAssets7z7zaArm64Exe.Content()
		}
		if err := os.WriteFile(zipExePath, data, 0644); err != nil {
			return err
		}
	}
	cmd := exec.Command(zipExePath, "x", filePath, "-o"+targetDir, "-aoa", "-bb0")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	logger.Infof("run extractor: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("7za extract failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func readLocalHeliumVersion(sd *SettingsData) string {
	parentPath := getString(sd.installPath)
	markerPath := filepath.Join(parentPath, heliumVersionMarker)
	if data, err := os.ReadFile(markerPath); err == nil {
		if version := strings.TrimSpace(string(data)); version != "" {
			return version
		}
	}
	for _, exeName := range heliumExecutableNames() {
		if version := GetVersion(sd, exeName); version != "-" {
			return version
		}
	}
	return "-"
}

func writeLocalHeliumVersion(sd *SettingsData) {
	parentPath := getString(sd.installPath)
	_ = os.WriteFile(filepath.Join(parentPath, heliumVersionMarker), []byte(getString(sd.curVer)), 0644)
}

func heliumExecutableNames() []string {
	return []string{"helium.exe", "chrome.exe"}
}

func isHeliumExecutableName(name string) bool {
	for _, exeName := range heliumExecutableNames() {
		if strings.EqualFold(name, exeName) {
			return true
		}
	}
	return false
}

func findHeliumExecutable(dir string) string {
	for _, exeName := range heliumExecutableNames() {
		exePath := filepath.Join(dir, exeName)
		if fileExist(exePath) {
			return exePath
		}
	}
	return ""
}
