package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func baseScreen(win fyne.Window, data *SettingsData) fyne.CanvasObject {
	installPathHandle(data)
	folderEntry := widget.NewEntryWithData(data.installPath)
	folderEntry.OnChanged = func(path string) {
		installPathHandle(data)
	}
	showFolderPicker := func() {
		folderDialog := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err == nil && lu != nil {
				_ = data.installPath.Set(lu.Path())
			}
		}, win)
		currentPath, _ := data.installPath.Get()
		if currentPath != "" {
			if listableURI, err := storage.ListerForURI(storage.NewFileURI(currentPath)); err == nil {
				folderDialog.SetLocation(listableURI)
			}
		}
		folderDialog.Show()
	}
	folderBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), showFolderPicker)
	data.folderEntryStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.folderEntryStatus.Get(); b {
			folderEntry.Disable()
			folderBtn.Disable()
		} else {
			folderEntry.Enable()
			folderBtn.Enable()
		}
	}))
	checkBtn := widget.NewButtonWithIcon(LoadString("CheckBtnLabel"), theme.SearchIcon(), func() {
		err := syncHeliumInfo(data)
		if err != nil {
			alertInfo(LoadString("UpdateCheckErrorMsg"), win)
		}
	})
	createLnkBtn := widget.NewButtonWithIcon(LoadString("CreateLnkBtnLabel"), theme.ContentAddIcon(), func() {
		err := createDeskLnk(data)
		if err != nil {
			alertInfo(LoadString("CreateLnkFailMsg"), win)
		} else {
			alertInfo(LoadString("CreateLnkSuccessMsg"), win)
		}
	})
	downloadBtn = widget.NewButtonWithIcon(LoadString("InstallBtnLabel"), theme.DownloadIcon(), func() {
		ov, _ := data.oldVer.Get()
		cv, _ := data.curVer.Get()
		if cv == ov {
			alertInfo(LoadString("NoNeedUpdateMsg"), win)
		} else {
			parentPath, _ := data.installPath.Get()
			heliumInUse := isProcessExist(filepath.Join(parentPath, "helium.exe"))
			if heliumInUse {
				alertInfo(LoadString("HeliumRunningMsg"), win)
			} else if runFlag == 1 {
				alertInfo(LoadString("HeliumUpdateRunningMsg"), win)
			} else {
				runFlag = 1
				if getString(data.oldVer) == "-" {
					alertConfirm(LoadString("FirstInstallMsg"), func(b bool) {
						if b {
							execDownAndUnzip(data, downloadProgress, 0)
						} else {
							runFlag = 0
						}
					}, win)
				} else {
					execDownAndUnzip(data, downloadProgress, 1)
				}
			}
		}
	})
	data.downBtnStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.downBtnStatus.Get(); b {
			downloadBtn.Disable()
		} else {
			downloadBtn.Enable()
		}
	}))
	data.checkBtnStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.checkBtnStatus.Get(); b {
			checkBtn.Disable()
		} else {
			checkBtn.Enable()
		}
	}))
	buttons := container.NewHBox(folderBtn)
	bar := container.NewBorder(nil, nil, buttons, nil, folderEntry)
	curVerLabel := widget.NewLabelWithData(data.curVer)
	curVerLabel.TextStyle.Bold = true
	oldVer := readLocalHeliumVersion(data)
	logger.Info("helium version:", oldVer)
	_ = data.oldVer.Set(oldVer)
	form := widget.NewForm(
		&widget.FormItem{Text: LoadString("InstallLabel"), Widget: bar},
		&widget.FormItem{Text: LoadString("NowVerLabel"), Widget: widget.NewLabelWithData(data.oldVer)},
		&widget.FormItem{Text: LoadString("LatestVerLabel"), Widget: curVerLabel},
		&widget.FormItem{Text: LoadString("FileSizeLabel"), Widget: widget.NewLabelWithData(data.fileSize)},
	)
	downloadProgress = widget.NewProgressBar()
	downloadProgress.TextFormatter = func() string {
		fs, _ := data.fileSize.Get()
		if downloadErrorFlag.Load() {
			return LoadString("DownloadFailedMsg")
		} else if downloadProgress.Max*0.9 == downloadProgress.Value {
			return fmt.Sprintf(LoadString("DownLoadedProcessMsg"), fs)
		} else if downloadProgress.Max == downloadProgress.Value {
			return LoadString("InstalledMsg")
		} else if downloadProgress.Value == 0.95 {
			return LoadString("Download95Msg")
		}
		fsFloatStr := strings.Split(fs, " ")[0]
		fsFloat, err := strconv.ParseFloat(fsFloatStr, 64)
		if err != nil {
			return LoadString("DownloadNotStartedMsg")
		}
		return fmt.Sprintf(LoadString("DownloadingMsg"), fsFloat*downloadProgress.Value, fs)
	}
	data.processStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.processStatus.Get(); b {
			downloadProgress.Show()
		} else {
			downloadProgress.Hide()
		}
	}))
	if !getBool(data.autoUpdate) {
		go func() {
			err := syncHeliumInfo(data)
			if err != nil {
				alertInfo(LoadString("UpdateCheckErrorMsg"), win)
			}
		}()
	}
	logger.Debug("Base tab load success.")
	return container.New(&buttonLayout{}, form, container.NewVBox(downloadProgress, container.NewGridWithColumns(3, checkBtn, downloadBtn, createLnkBtn)))
}

func syncHeliumInfo(data *SettingsData) error {
	heliumInfo, err := getLatestHeliumInfo(data)
	if err != nil {
		return err
	}
	data.curVer.Set(heliumInfo.Version)
	data.fileSize.Set(formatFileSize(heliumInfo.Size))
	data.fileSizeRaw.Set(int(heliumInfo.Size))
	data.urlList.Set(heliumInfo.Urls)
	data.SHA1.Set("-")
	data.SHA256.Set("-")
	data.downBtnStatus.Set(false)
	return nil
}

func execDownAndUnzip(data *SettingsData, downloadProgress *widget.ProgressBar, installType int) {
	data.checkBtnStatus.Set(true)
	data.folderEntryStatus.Set(true)
	data.processStatus.Set(true)
	installRoot := installRootFromPath(getString(data.installPath))
	if installType == 0 {
		initInstallDirs(data)
	}
	appPath := getString(data.installPath)
	url := getDownloadUrl(data.urlList)
	downloadErrorFlag.Store(false)
	downloadProgress.SetValue(0)
	fileName := filepath.Join(installRoot, getFileName(url))

	dl := NewDownloader(data, url, fileName, 16, downloadProgress)
	if fs, _ := data.fileSizeRaw.Get(); fs > 0 {
		dl.FileSize = int64(fs)
	}
	dl.UseProxy = getBool(data.downloadChromeViaProxy)

	go func() {
		err := <-dl.Done
		if err != nil {
			logger.Errorf("Helium download failed: %v", err)
			downloadErrorFlag.Store(true)
			fyne.DoAndWait(func() { downloadProgress.SetValue(0) })
			defer data.checkBtnStatus.Set(false)
			defer data.folderEntryStatus.Set(false)
			defer func() { runFlag = 0 }()
			return
		}

		if !fileExist(fileName) {
			downloadErrorFlag.Store(true)
			fyne.DoAndWait(func() { downloadProgress.SetValue(0) })
		} else {
			fyne.DoAndWait(func() { downloadProgress.SetValue(0.95) })
			err := installHeliumPackage(fileName, appPath)
			if err != nil {
				logger.Errorf("Helium install failed: %v", err)
				downloadErrorFlag.Store(true)
				fyne.DoAndWait(func() { downloadProgress.SetValue(0) })
			} else {
				if !getBool(data.remainInstallFileSettings) {
					_ = os.Remove(fileName)
				}
				if !getBool(data.remainHistoryFileSettings) {
					_ = os.RemoveAll(filepath.Join(appPath, getString(data.oldVer)))
				}
				fyne.DoAndWait(func() { downloadProgress.SetValue(1) })
				writeLocalHeliumVersion(data)
				data.oldVer.Set(getString(data.curVer))
			}
		}
		defer data.checkBtnStatus.Set(false)
		defer data.folderEntryStatus.Set(false)
		defer func() { runFlag = 0 }()
	}()

	dl.Start()
}

func createDeskLnk(data *SettingsData) error {
	parentPath, _ := data.installPath.Get()
	exePath := filepath.Join(parentPath, "helium.exe")
	if fileExist(exePath) {
		desktopPath, err := GetDesktopPath()
		if err != nil {
			logger.Debug(err)
			return err
		}
		logger.Debug("Desktop Path:", desktopPath)
		linkPath := filepath.Join(desktopPath, "Helium.lnk")
		err = makeLink(exePath, linkPath)
		if err != nil {
			logger.Debug(err)
		}
		return err
	}
	return errors.New("executable file not found")
}

func initInstallDirs(data *SettingsData) {
	root := installRootFromPath(getString(data.installPath))
	_ = os.MkdirAll(filepath.Join(root, "Application"), os.ModePerm)
	_ = os.MkdirAll(filepath.Join(root, "Cache"), os.ModePerm)
	_ = os.MkdirAll(filepath.Join(root, "Data"), os.ModePerm)
	_ = data.installPath.Set(filepath.Join(root, "Application"))
}

func installRootFromPath(path string) string {
	root := strings.TrimSpace(path)
	if strings.EqualFold(filepath.Base(root), "Application") {
		return filepath.Dir(root)
	}
	return root
}

var (
	downloadProgress  *widget.ProgressBar
	downloadBtn       *widget.Button
	downloadErrorFlag atomic.Bool
)

func getDownloadUrl(list binding.StringList) string {
	urls, _ := list.Get()
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func installPathHandle(data *SettingsData) {
	dir := strings.TrimSpace(getString(data.installPath))
	if dir == "" {
		_ = data.oldVer.Set("-")
		return
	}
	dir = filepath.Clean(dir)
	if !dirExist(dir) {
		_ = data.oldVer.Set("-")
		return
	}
	if appDir := filepath.Join(dir, "Application"); fileExist(filepath.Join(appDir, "helium.exe")) {
		dir = appDir
		_ = data.installPath.Set(dir)
	}

	dirHandle, err := os.Open(dir)
	if err != nil {
		logger.Warnf("open install path failed: %s, err=%v", dir, err)
		_ = data.oldVer.Set("-")
		return
	}
	defer dirHandle.Close()
	fileInfos, err := dirHandle.Readdir(-1)
	if err != nil {
		logger.Warnf("read install path failed: %s, err=%v", dir, err)
		_ = data.oldVer.Set("-")
		return
	}
	result := false
	v := ""
	for _, fileInfo := range fileInfos {
		name := fileInfo.Name()
		if strings.EqualFold(name, "helium.exe") {
			result = true
		}
		if fileInfo.IsDir() && isNumeric(strings.ReplaceAll(name, ".", "")) {
			v = fileInfo.Name()
		}
	}
	if result {
		data.installPath.Set(dir)
		if version := readLocalHeliumVersion(data); version != "-" {
			data.oldVer.Set(version)
		} else {
			data.oldVer.Set(v)
		}
	} else {
		data.oldVer.Set("-")
	}
	if getBool(data.downBtnStatus) {
		data.checkBtnStatus.Set(false)
	}
}

func defaultInstallPath() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		dir := filepath.Join(localAppData, "imput", "Helium", "Application")
		if dirExist(dir) {
			return dir
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}
