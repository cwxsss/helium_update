package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/img/helium_updater.png
var heliumUpdaterIconBytes []byte

var resourceAssetsImgHeliumUpdaterPng = fyne.NewStaticResource("helium_updater.png", heliumUpdaterIconBytes)
