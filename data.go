package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"fyne.io/fyne/v2/data/binding"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

func getConfigPath() string {
	appDataDir := os.Getenv("APPDATA")
	if appDataDir == "" {
		appDataDir = os.TempDir()
	}
	ex, err := os.Executable()
	if err != nil {
		logger.Error(err)
	}
	exPath := filepath.Dir(ex)
	files, _ := filepath.Glob(filepath.Join(exPath, "*"))
	if slices.Contains(files, filepath.Join(exPath, "config.json")) {
		return filepath.Join(exPath, "config.json")
	} else {
		return filepath.Join(appDataDir, "helium_updater", "config.json")
	}
}

// 初始化数据
func initData() *SettingsData {
	configFilePath := getConfigPath()
	logger.Debugf("Config path: %s", configFilePath)
	configFileExist := fileExist(configFilePath)
	var config Config
	sd := createSettings()
	if configFileExist {
		file, err := os.Open(configFilePath)
		if err != nil {
			logger.Errorln("无法打开文件:", err)
		}
		decoder := jsoniter.NewDecoder(file)
		if err = decoder.Decode(&config); err != nil {
			logger.Errorln("解析 JSON 失败:", err)
		}
		logger.Debug(zap.Any("config", config))
		currentUpdaterDir := defaultInstallPath()
		if config.InstallPath != "" && samePath(config.UpdaterDir, currentUpdaterDir) {
			sd.installPath.Set(config.InstallPath)
		}
		if config.DataPath != "" {
			sd.dataPath.Set(config.DataPath)
		}
		if config.CachePath != "" {
			sd.cachePath.Set(config.CachePath)
		}
		sd.branch.Set(config.VersionBranch)
		if config.HeliumPackageType != "" {
			sd.heliumPackageType.Set(config.HeliumPackageType)
		}
		sd.urlKey.Set(config.DownloadChannel)
		sd.remainInstallFileSettings.Set(config.RemainInstallFile)
		sd.remainHistoryFileSettings.Set(config.RemainHistoryFile)
		sd.oldPlusVer.Set(config.OldPlusVer)
		sd.chromePlus.Set(config.ChromePlus)
		sd.themeSettings.Set(config.Theme)
		sd.langSettings.Set(config.Lang)
		sd.ghProxy.Set(config.GhProxy)
		sd.proxyType.Set(config.ProxyType)
		sd.downloadChromeViaProxy.Set(config.DownloadChromeViaProxy)
		sd.autoUpdate.Set(config.AutoUpdate)
	}
	return sd
}

func saveConfig(data *SettingsData) error {
	config := Config{
		InstallPath:            getString(data.installPath),
		UpdaterDir:             defaultInstallPath(),
		DataPath:               getString(data.dataPath),
		CachePath:              getString(data.cachePath),
		VersionBranch:          getString(data.branch),
		HeliumPackageType:      getString(data.heliumPackageType),
		DownloadChannel:        getString(data.urlKey),
		RemainInstallFile:      getBool(data.remainInstallFileSettings),
		RemainHistoryFile:      getBool(data.remainHistoryFileSettings),
		OldPlusVer:             getString(data.oldPlusVer),
		ChromePlus:             getString(data.chromePlus),
		Theme:                  getString(data.themeSettings),
		Lang:                   getString(data.langSettings),
		GhProxy:                getString(data.ghProxy),
		ProxyType:              getString(data.proxyType),
		DownloadChromeViaProxy: getBool(data.downloadChromeViaProxy),
		AutoUpdate:             getBool(data.autoUpdate),
	}
	jsonData, _ := jsoniter.Marshal(config)
	configFilePath := getConfigPath()
	_ = os.Remove(configFilePath)
	configFileExist := fileExist(configFilePath)
	if !configFileExist {
		dir := filepath.Dir(configFilePath)
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			fmt.Println("无法创建目录:", err)
			return err
		}
		// 创建文件
		file, err := os.Create(configFilePath)
		if err != nil {
			fmt.Println("无法创建文件:", err)
			return err
		}
		defer file.Close()
		_, err = file.Write(jsonData)
		if err != nil {
			fmt.Println("无法写入文件:", err)
			return err
		}
	}
	return nil
}

// 创建配置数据
func createSettings() *SettingsData {
	installPath := binding.NewString()
	dataPath := binding.NewString()
	cachePath := binding.NewString()
	branch := binding.NewString()
	heliumPackageType := binding.NewString()
	oldVer := binding.NewString()
	oldVer.Set("-")
	curVer := binding.NewString()
	curVer.Set("-")
	fileSize := binding.NewString()
	fileSize.Set("-")
	fileSizeRaw := binding.NewInt()
	SHA1 := binding.NewString()
	SHA1.Set("-")
	SHA256 := binding.NewString()
	SHA256.Set("-")
	urlList := binding.NewStringList()
	_ = installPath.Set(defaultInstallPath())
	_ = dataPath.Set(defaultDataPath())
	_ = cachePath.Set(defaultCachePath())
	_ = branch.Set("stable")
	_ = heliumPackageType.Set(heliumPackageTypeExe)
	downBtnStatus := binding.NewBool()
	downBtnStatus.Set(true) // 初始下载按钮状态
	checkBtnStatus := binding.NewBool()
	checkBtnStatus.Set(true) // 初始检查按钮状态
	folderEntryStatus := binding.NewBool()
	folderEntryStatus.Set(false) //初始化Chrome安装目录状态
	urlKey := binding.NewString()
	urlKey.Set("github.com")
	processStatus := binding.NewBool()
	processStatus.Set(false) //初始化下载安装进度的进度条状态
	remainInstallFileSettings := binding.NewBool()
	remainInstallFileSettings.Set(false) //保留安装文件
	remainHistoryFileSettings := binding.NewBool()
	remainHistoryFileSettings.Set(false) //保留历史文件
	themeSettings := binding.NewString()
	themeSettings.Set(LoadString("SystemOption"))
	langSettings := binding.NewString()
	langSettings.Set(LoadString("SystemOption"))
	oldPlusVer := binding.NewString()
	curPlusVer := binding.NewString()
	curPlusVer.Set("-")
	oldPlusVer.Set("-")
	chromePlus := binding.NewString()
	chromePlus.Set("cwxsss/helium_plus")
	plusDownloadUrl := binding.NewString()
	plusFileSizeRaw := binding.NewInt()
	plusBtnStatus := binding.NewBool()
	plusBtnStatus.Set(true)
	plusProcessStatus := binding.NewBool()
	plusProcessStatus.Set(false)
	ghProxy := binding.NewString()
	ghProxy.Set("")
	proxyType := binding.NewString()
	proxyType.Set(proxyTypeHTTP)
	downloadChromeViaProxy := binding.NewBool()
	autoUpdate := binding.NewBool()
	autoUpdate.Set(false)
	return &SettingsData{
		installPath:               installPath,
		dataPath:                  dataPath,
		cachePath:                 cachePath,
		oldVer:                    oldVer,
		branch:                    branch,
		heliumPackageType:         heliumPackageType,
		curVer:                    curVer,
		fileSize:                  fileSize,
		fileSizeRaw:               fileSizeRaw,
		SHA1:                      SHA1,
		SHA256:                    SHA256,
		urlList:                   urlList,
		downBtnStatus:             downBtnStatus,
		checkBtnStatus:            checkBtnStatus,
		folderEntryStatus:         folderEntryStatus,
		urlKey:                    urlKey,
		processStatus:             processStatus,
		oldPlusVer:                oldPlusVer,
		curPlusVer:                curPlusVer,
		chromePlus:                chromePlus,
		plusDownloadUrl:           plusDownloadUrl,
		plusFileSizeRaw:           plusFileSizeRaw,
		plusBtnStatus:             plusBtnStatus,
		plusProcessStatus:         plusProcessStatus,
		remainInstallFileSettings: remainInstallFileSettings,
		remainHistoryFileSettings: remainHistoryFileSettings,
		themeSettings:             themeSettings,
		langSettings:              langSettings,
		ghProxy:                   ghProxy,
		proxyType:                 proxyType,
		downloadChromeViaProxy:    downloadChromeViaProxy,
		autoUpdate:                autoUpdate,
	}
}
