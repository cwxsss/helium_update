package main

import (
	"strings"
	"testing"
)

func TestSystemVersionDLLPathUsesSystemDirectory(t *testing.T) {
	got := systemVersionDLLPath(`C:\Windows`)
	want := `C:\Windows\System32\version.dll`
	if got != want {
		t.Fatalf("systemVersionDLLPath() = %q, want %q", got, want)
	}
}

func TestChromeExecutablePathUsesChromeExe(t *testing.T) {
	got := chromeExecutablePath(`D:\software\Helium\Application`)
	want := `D:\software\Helium\Application\chrome.exe`
	if got != want {
		t.Fatalf("chromeExecutablePath() = %q, want %q", got, want)
	}
}

func TestExecutableNameMatchesIgnoresCase(t *testing.T) {
	if !executableNameMatches("CHROME.EXE", "chrome.exe") {
		t.Fatal("executableNameMatches() should match chrome.exe without case sensitivity")
	}
}

func TestVersionReplacementScriptOnlyMovesVersionDLL(t *testing.T) {
	script := versionReplacementScript(
		`C:\temp\version.dll`,
		`D:\Helium\Application\version.dll`,
		`D:\Helium\Application\Helium++_v1.18.1_x64.7z`,
		`D:\Helium\Application\.chrome_plus_extract`,
		`D:\Helium\Application\helium_updater.exe`,
	)
	if !strings.Contains(script, `set "source=C:\temp\version.dll"`) ||
		!strings.Contains(script, `set "target=D:\Helium\Application\version.dll"`) ||
		!strings.Contains(script, `move /Y "%source%" "%target%"`) {
		t.Fatal("replacement script does not move version.dll to the installation directory")
	}
	if strings.Contains(strings.ToLower(script), "chrome++.ini") {
		t.Fatal("replacement script must not modify chrome++.ini")
	}
}
