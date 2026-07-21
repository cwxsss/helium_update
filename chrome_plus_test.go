package main

import (
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

func TestPendingVersionDLLPathKeepsExistingDLLUntouched(t *testing.T) {
	got := pendingVersionDLLPath(`D:\Helium\Application`)
	want := `D:\Helium\Application\version.dll.new`
	if got != want {
		t.Fatalf("pendingVersionDLLPath() = %q, want %q", got, want)
	}
}
