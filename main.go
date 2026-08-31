//go:build !mono

// Package main is the entry point for the Pickleball Theme plugin.
package main

import (
	sdkapi "sdk/api"

	"com.flarego.pickleball-theme/app"
	"com.flarego.pickleball-theme/app/navs"
	"com.flarego.pickleball-theme/app/theme"
)

func main() {}

func Init(api sdkapi.IPluginApi) error {
	app.SetupRoutes(api)
	navs.SetAdminNavs(api)
	theme.SetPortalTheme(api)
	return nil
}
