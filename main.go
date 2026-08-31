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
			Theme:               windows.SystemDefault,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:           windows.RGB(24, 28, 36),
				DarkModeTitleBarInactive:   windows.RGB(29, 34, 43),
				DarkModeTitleText:          windows.RGB(236, 240, 246),
				DarkModeTitleTextInactive:  windows.RGB(174, 182, 196),
				DarkModeBorder:             windows.RGB(52, 59, 73),
				DarkModeBorderInactive:     windows.RGB(43, 49, 60),
				LightModeTitleBar:          windows.RGB(248, 250, 252),
				LightModeTitleBarInactive:  windows.RGB(242, 245, 249),
				LightModeTitleText:         windows.RGB(32, 38, 52),
				LightModeTitleTextInactive: windows.RGB(114, 123, 139),
				LightModeBorder:            windows.RGB(223, 228, 236),
				LightModeBorderInactive:    windows.RGB(229, 232, 239),
			},
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
