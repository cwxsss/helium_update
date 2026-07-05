package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	jsoniter "github.com/json-iterator/go"
)

func chromePlusScreen(win fyne.Window, data *SettingsData) fyne.CanvasObject {
	var githubReleaseMap map[string]GithubRelease
	var versionList []string
	var downBtn *widget.Button
	var versionSelect *widget.Select
	plusSourceRepos := map[string]string{
		"cwxsss/helium_plus":   "cwxsss/helium_plus",
		"Bush2021/chrome_plus": "Bush2021/chrome_plus",
	}
	selectedSource := normalizeChromePlusSource(getString(data.chromePlus))
	_ = data.chromePlus.Set(selectedSource)
	chromePlusRadio := widget.NewRadioGroup([]string{
		"cwxsss/helium_plus",
		"Bush2021/chrome_plus",
	}, func(value string) {
		data.chromePlus.Set(value)
		data.curPlusVer.Set("-")
		data.plusDownloadUrl.Set("")
		data.plusFileSizeRaw.Set(0)
		versionSelect.ClearSelected()
		versionSelect.SetOptions([]string{})
		versionSelect.Disable()
		downBtn.Disable()
	})
	versionSelect = widget.NewSelect([]string{}, func(ver string) {
		setPlusVer(data, ver, githubReleaseMap)
	})
	versionSelect.PlaceHolder = LoadString("VersionSelectPlaceHolder")
	versionSelect.Disable()
	chromePlusRadio.Selected = selectedSource
	downBtn = widget.NewButtonWithIcon(LoadString("InstallBtnLabel"), theme.DownloadIcon(), func() {
		ov, _ := data.oldPlusVer.Get()
		cv, _ := data.curPlusVer.Get()
		if cv == ov && fileExist(path.Join(getString(data.installPath), "version.dll")) {
			alertInfo(LoadString("NoNeedUpdateMsg"), win)
		} else {
			parentPath, _ := data.installPath.Get()
			heliumInUse := isProcessExist(filepath.Join(parentPath, "helium.exe"))
			if heliumInUse {
				alertInfo(LoadString("HeliumRunningMsg"), win)
			} else {
				installPlus(data, win)
			}
		}
	})
	checkBtn := widget.NewButtonWithIcon(LoadString("CheckBtnLabel"), theme.SearchIcon(), func() {
		var err error
		chromePlusRadio.Disable()
		checkBtn.Disable()
		checkBtn.SetText(LoadString("CheckBtnLabel") + "...")
		data.curPlusVer.Set("-")
		downBtn.Disable()
		githubReleaseMap, versionList, err = getChromePlusInfo(data, plusSourceRepos[getString(data.chromePlus)])
		chromePlusRadio.Enable()
		checkBtn.Enable()
		checkBtn.SetText(LoadString("CheckBtnLabel"))
		if err != nil {
			alertInfo(LoadString("UpdateErrMsg"), win)
		} else {
			if githubReleaseMap != nil && len(versionList) > 0 {
				versionSelect.SetOptions(versionList)
				versionSelect.SetSelected(versionList[0])
				setPlusVer(data, versionList[0], githubReleaseMap)
				versionSelect.Enable()
				downBtn.Enable()
			} else {
				alertInfo(LoadString("UpdateErrMsg"), win)
			}
		}
	})
	data.plusBtnStatus.AddListener(binding.NewDataListener(func() {
		if getBool(data.plusBtnStatus) {
			downBtn.Disable()
		} else {
			downBtn.Enable()
		}
	}))
	curVerLabel := widget.NewLabelWithData(data.curPlusVer)
	curVerLabel.TextStyle.Bold = true
	oldPlusVer := GetVersion(data, "version.dll")
	logger.Info("chrome++ version:", oldPlusVer)
	_ = data.oldPlusVer.Set(oldPlusVer)
	form := widget.NewForm(
		&widget.FormItem{Text: LoadString("NowVerLabel"), Widget: widget.NewLabelWithData(data.oldPlusVer)},
		&widget.FormItem{Text: LoadString("LatestVerLabel"), Widget: curVerLabel},
		&widget.FormItem{Text: LoadString("VersionSelectLabel"), Widget: versionSelect},
		&widget.FormItem{Text: LoadString("PlusSourceLabel"), Widget: chromePlusRadio},
	)
	rich := widget.NewRichTextFromMarkdown(LoadString("MarkdownMsg"))
	rich.Wrapping = fyne.TextWrapWord
	infoCard := widget.NewCard("", "", rich)
	plusDownloadProgress = widget.NewProgressBar()
	plusDownloadProgress.TextFormatter = func() string {
		if plusDownloadError.Load() {
			return LoadString("DownloadFailedMsg")
		} else if plusDownloadProgress.Max*0.9 == plusDownloadProgress.Value {
			return LoadString("PlusDownloadedMsg")
		} else if plusDownloadProgress.Max == plusDownloadProgress.Value {
			return LoadString("InstalledMsg")
		}
		return LoadString("PlusDownloadingMsg")
	}
	data.plusProcessStatus.AddListener(binding.NewDataListener(func() {
		if getBool(data.plusProcessStatus) {
			plusDownloadProgress.Show()
		} else {
			plusDownloadProgress.Hide()
		}
	}))
	logger.Debug("Chrome++ tab load success.")
	return container.New(&buttonLayout{}, container.NewVBox(form,
		infoCard,
	), container.NewVBox(plusDownloadProgress, container.NewGridWithColumns(2, checkBtn, downBtn)))
}

func normalizeChromePlusSource(source string) string {
	switch source {
	case "Bush2021", "Bush2021/chrome_plus":
		return "Bush2021/chrome_plus"
	case "cwxsss", "cwxsss/helium_plus":
		return "cwxsss/helium_plus"
	default:
		return "cwxsss/helium_plus"
	}
}

func setPlusVer(data *SettingsData, ver string, releaseMap map[string]GithubRelease) {
	plusInfo := releaseMap[ver]
	data.curPlusVer.Set(plusInfo.TagName)
	for _, asset := range plusInfo.Assets {
		name := strings.ToLower(asset.Name)
		if isChromePlusAssetName(name) && (strings.Contains(name, "x64") || strings.Contains(name, "amd64")) {
			data.plusDownloadUrl.Set(asset.BrowserDownloadURL)
			data.plusFileSizeRaw.Set(int(asset.Size))
			return
		}
	}
	for _, asset := range plusInfo.Assets {
		if isChromePlusAssetName(strings.ToLower(asset.Name)) {
			data.plusDownloadUrl.Set(asset.BrowserDownloadURL)
			data.plusFileSizeRaw.Set(int(asset.Size))
			return
		}
	}
}

var (
	plusDownloadProgress *widget.ProgressBar
	plusDownloadError    atomic.Bool
)

func installPlus(data *SettingsData, win fyne.Window) {
	url := getString(data.plusDownloadUrl)
	data.plusBtnStatus.Set(true)
	plusDownloadError.Store(false)
	plusDownloadProgress.SetValue(0)
	data.plusProcessStatus.Set(true)
	plusArch := "x64"
	parentPath, _ := data.installPath.Get()
	fileName := getFileName(url)
	fileName = filepath.Join(parentPath, fileName)

	dl := NewDownloader(data, url, fileName, 16, plusDownloadProgress)
	if fs, _ := data.plusFileSizeRaw.Get(); fs > 0 {
		dl.FileSize = int64(fs)
	}

	go func() {
		err := <-dl.Done
		if err != nil {
			logger.Errorf("Chrome++ 下载失败: %v", err)
			plusDownloadError.Store(true)
			fyne.DoAndWait(func() { plusDownloadProgress.SetValue(0) })
			data.plusProcessStatus.Set(false)
			data.plusBtnStatus.Set(false)
			return
		}

		UnCompress7zFilter(fileName, parentPath, plusArch)
		os.Rename(filepath.Join(parentPath, plusArch, "App", "version.dll"), path.Join(parentPath, "version.dll"))
		if !fileExist(path.Join(parentPath, "chrome++.ini")) {
			os.Rename(filepath.Join(parentPath, plusArch, "App", "chrome++.ini"), path.Join(parentPath, "chrome++.ini"))
		}
		os.Remove(fileName)
		os.RemoveAll(filepath.Join(parentPath, plusArch))
		fyne.DoAndWait(func() { plusDownloadProgress.SetValue(1) })
		defer data.oldPlusVer.Set(getString(data.curPlusVer))
		defer data.plusProcessStatus.Set(false)
		defer data.plusBtnStatus.Set(false)
		alertInfo(LoadString("InstalledMsg"), win)
	}()

	dl.Start()
}
func setProxy(sd *SettingsData, reqUrl string) (*http.Client, string) {
	return newHTTPClientWithProxy(sd, 5*time.Second), rewriteWithGithubProxy(sd, reqUrl)
}

func getChromePlusInfo(sd *SettingsData, repo string) (map[string]GithubRelease, []string, error) {
	if repo == "" {
		repo = "cwxsss/helium_plus"
	}
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=10", repo)
	client, reqUrl := setProxy(sd, apiUrl)
	response, err := client.Get(reqUrl)
	if err != nil {
		logger.Errorln(err)
		return nil, nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Errorln(err)
		return nil, nil, err
	}
	var githubReleases []GithubRelease
	if err = jsoniter.Unmarshal(data, &githubReleases); err != nil {
		logger.Errorln(err)
		return nil, nil, err
	}
	result := make(map[string]GithubRelease)
	versionList := make([]string, 0)
	for _, item := range githubReleases {
		if !hasChromePlusAsset(item) {
			continue
		}
		result[item.TagName] = item
		versionList = append(versionList, item.TagName)
	}
	logger.Debugf("Chrome++ source:%s versions:%v", repo, versionList)
	return result, versionList, err
}

func hasChromePlusAsset(release GithubRelease) bool {
	for _, asset := range release.Assets {
		if isChromePlusAssetName(strings.ToLower(asset.Name)) {
			return true
		}
	}
	return false
}

func isChromePlusAssetName(name string) bool {
	return strings.HasSuffix(name, ".7z") &&
		(strings.Contains(name, "helium++") ||
			strings.Contains(name, "chrome++") ||
			strings.Contains(name, "chrome_plus"))
}
