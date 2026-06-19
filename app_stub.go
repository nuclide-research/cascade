//go:build !webview

package main

// nativeApp is the default (no-CGo) standalone-window implementation: it serves
// the UI and opens it as a frameless window via an installed Chromium browser
// (--app=). Build with `-tags webview` to get a true OS webview window instead
// (see app_webview.go), which needs CGo + a system webview library.
func nativeApp() error {
	return serveApp()
}
