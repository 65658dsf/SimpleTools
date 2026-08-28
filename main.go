package main

import (
	"embed"

	"github.com/65658dsf/SimpleTools/internal/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// version and updatePublicKey are replaced by release builds with ldflags.
var version = "0.1.0"
var updatePublicKey = ""

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	application := app.New(version, updatePublicKey)
	err := wails.Run(&options.App{
		Title:     "SimpleTools",
		Width:     1180,
		Height:    780,
		MinWidth:  960,
		MinHeight: 620,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 255},
		OnStartup:        application.Startup,
		OnShutdown:       application.Shutdown,
		Bind:             []interface{}{application},
		Windows: &windows.Options{
			WebviewUserDataPath: "SimpleTools",
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
		},
		DragAndDrop: &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true},
	})
	if err != nil {
		panic(err)
	}
}
