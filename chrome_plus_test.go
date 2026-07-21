package main

import "testing"

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
