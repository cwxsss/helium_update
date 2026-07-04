package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/robfig/cron/v3"
)

var cronManager = cron.New(cron.WithSeconds())

func chromeAutoUpdate(a fyne.App, win fyne.Window, data *SettingsData) {
	if desk, ok := a.(desktop.App); ok {
		addUpdateCron(data)
		updateMenu := fyne.NewMenuItem(LoadString("SystemTrayAutoUpdateMenu"), func() {
			_ = data.autoUpdate.Set(!getBool(data.autoUpdate))
		})
		updateMenu.Checked = getBool(data.autoUpdate)
		if getBool(data.autoUpdate) {
			cronManager.Start()
		} else {
			cronManager.Stop()
		}
		m := fyne.NewMenu("",
			updateMenu,
			fyne.NewMenuItem(LoadString("SystemTrayShowMenu"), func() {
				win.Show()
			}),
			fyne.NewMenuItem(LoadString("SystemTrayHideMenu"), func() {
				win.Hide()
			}),
		)
		data.autoUpdate.AddListener(binding.NewDataListener(func() {
			updateMenu.Checked = getBool(data.autoUpdate)
			if getBool(data.autoUpdate) {
				cronManager.Start()
			} else {
				cronManager.Stop()
			}
			m.Refresh()
		}))
		desk.SetSystemTrayMenu(m)
	}
	logger.Debug("Set system tray menu success.")
}

var runFlag = 0

var currentData *SettingsData

func addUpdateCron(data *SettingsData) {
	currentData = data
	spec := "0 0 0/1 * * ?"
	_, _ = cronManager.AddFunc(spec, func() {
		parentPath, _ := data.installPath.Get()
		heliumInUse := isProcessExist(filepath.Join(parentPath, "helium.exe"))
		if runFlag == 1 || heliumInUse {
			return
		}
		if getString(data.oldVer) != "-" {
			runFlag = 1
			heliumInfo, err := getLatestHeliumInfo(data)
			if err != nil {
				runFlag = 0
				return
			}
			_ = data.curVer.Set(heliumInfo.Version)
			_ = data.fileSize.Set(formatFileSize(heliumInfo.Size))
			_ = data.urlList.Set(heliumInfo.Urls)
			_ = data.downBtnStatus.Set(false)
			oldVer := readLocalHeliumVersion(data)
			logger.Info("helium version:", oldVer)
			_ = data.oldVer.Set(oldVer)
			ov, _ := data.oldVer.Get()
			cv, _ := data.curVer.Get()
			if cv != ov {
				data.fileSizeRaw.Set(int(heliumInfo.Size))
				autoInstall(data)
			}
			runFlag = 0
			downloadBtn.SetText(LoadString("InstallBtnLabel"))
		}
	})
}

func timeCost(start time.Time) {
	tc := time.Since(start)
	fmt.Printf("time cost = %v\n", tc)
}

func autoInstall(data *SettingsData) {
	url := getDownloadUrl(data.urlList)
	parentPath, _ := data.installPath.Get()
	fileName := filepath.Join(parentPath, getFileName(url))
	if !fileExist(fileName) {
		if !downloadHelium(url, fileName) {
			return
		}
	}
	defer timeCost(time.Now())
	if err := installHeliumPackage(fileName, parentPath); err != nil {
		logger.Errorf("Automatic Helium update failed: %v", err)
		return
	}
	if !getBool(data.remainInstallFileSettings) {
		_ = os.Remove(fileName)
	}
	if !getBool(data.remainHistoryFileSettings) {
		_ = os.RemoveAll(filepath.Join(parentPath, getString(data.oldVer)))
	}
	writeLocalHeliumVersion(data)
	_ = data.oldVer.Set(getString(data.curVer))
}

func downloadHelium(url, fileName string) bool {
	autoDownloadProgress := widget.NewProgressBar()
	autoDownloadProgress.SetValue(0)
	autoDownloadProgress.TextFormatter = func() string {
		percentageStr := fmt.Sprintf("%.1f%%", autoDownloadProgress.Value*100.0/0.9)
		downloadBtn.SetText(LoadString("AutoUpdateProgress") + percentageStr)
		return ""
	}

	dl := NewDownloader(currentData, url, fileName, 16, autoDownloadProgress)
	if currentData != nil {
		if fs, _ := currentData.fileSizeRaw.Get(); fs > 0 {
			dl.FileSize = int64(fs)
		}
		dl.UseProxy = getBool(currentData.downloadChromeViaProxy)
	}
	dl.Start()
	err := <-dl.Done
	if err != nil {
		logger.Errorf("Automatic Helium download failed: %v", err)
		return false
	}
	return fileExist(fileName)
}
