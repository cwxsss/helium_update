package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	chromePlusRadio := widget.NewRadioGroup([]string{
		"Bush2021"}, func(value string) {
		data.chromePlus.Set(value)
	})
	versionSelect := widget.NewSelect([]string{}, func(ver string) {
		setPlusVer(data, ver, githubReleaseMap)
	})
	versionSelect.PlaceHolder = LoadString("VersionSelectPlaceHolder")
	versionSelect.Disable()
	chromePlusRadio.Selected = getString(data.chromePlus)
	chromePlusRadio.Disable()
	downBtn := widget.NewButtonWithIcon(LoadString("InstallBtnLabel"), theme.DownloadIcon(), func() {
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
		githubReleaseMap, versionList, err = getChromePlusInfo(data)
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
		&widget.FormItem{Text: LoadString("LatestVerLabel"), Widget: versionSelect},
		&widget.FormItem{Text: LoadString("BranchLabel"), Widget: chromePlusRadio},
	)
	rich := widget.NewRichTextFromMarkdown(LoadString("MarkdownMsg"))
	rich.Wrapping = fyne.TextWrapWord
	infoCard := widget.NewCard("", "", rich)
	plusDownloadProgress = widget.NewProgressBar()
	plusDownloadProgress.TextFormatter = func() string {
		if plusDownloadError.Load() {
			return "下载失败，请稍后重试"
		} else if plusDownloadProgress.Max*0.9 == plusDownloadProgress.Value {
			return fmt.Sprintf(LoadString("PlusDownloadedMsg"))
		} else if plusDownloadProgress.Max == plusDownloadProgress.Value {
			return "安装完成"
		}
		return fmt.Sprintf(LoadString("PlusDownloadingMsg"))
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

func setPlusVer(data *SettingsData, ver string, releaseMap map[string]GithubRelease) {
	plusInfo := releaseMap[ver]
	data.curPlusVer.Set(plusInfo.TagName)
	for _, asset := range plusInfo.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, "x64") || strings.Contains(name, "amd64") {
			data.plusDownloadUrl.Set(asset.BrowserDownloadURL)
			data.plusFileSizeRaw.Set(int(asset.Size))
			return
		}
	}
	if len(plusInfo.Assets) > 0 {
		data.plusDownloadUrl.Set(plusInfo.Assets[0].BrowserDownloadURL)
		data.plusFileSizeRaw.Set(int(plusInfo.Assets[0].Size))
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
	ghProxy := getString(sd.ghProxy)
	client := http.Client{Timeout: time.Second * time.Duration(5), Transport: &http.Transport{
		Proxy: GetProxyURL(),
	}}
	if ghProxy != "" {
		if getString(sd.proxyType) == "GH-PROXY" {
			reqUrl = pathJoin(ghProxy, reqUrl)
		} else {
			if getString(sd.proxyType) == "HTTP(S)" && !strings.HasPrefix(ghProxy, "http") {
				ghProxy = "http://" + ghProxy
			} else if getString(sd.proxyType) == "SOCKS5" && !strings.HasPrefix(ghProxy, "socks5") {
				ghProxy = "socks5://" + ghProxy
			}
			urli := url.URL{}
			urlproxy, _ := urli.Parse(ghProxy)
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(urlproxy),
			}
		}
	}
	return &client, reqUrl
}

func getChromePlusInfo(sd *SettingsData) (map[string]GithubRelease, []string, error) {
	apiUrl := "https://api.github.com/repos/Bush2021/chrome_plus/releases?per_page=10"
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
		result[item.TagName] = item
		versionList = append(versionList, item.TagName)
	}
	logger.Debugf("Chrome++ versions:%v", versionList)
	return result, versionList, err
}
