//go:build webview

package main

// Build:  go build -tags webview -o cascade .
//
// Requires CGo and a system webview library:
//   Linux:   sudo apt install libwebkit2gtk-4.1-dev   (or 4.0-dev on older distros)
//            CGO_ENABLED=1 go build -tags webview -o cascade .
//   Windows: the Microsoft Edge WebView2 runtime ships with Windows 10/11.
//            Build on Windows (or cross-compile with a mingw toolchain).
//   macOS:   uses the system WKWebView; no extra install.
//
// This reuses the exact same embedded SSE server and HTML UI as the browser and
// --app modes; only the window host differs.

import (
	"fmt"

	webview "github.com/webview/webview_go"
)

// nativeApp runs the UI in a true native OS webview window. The HTTP server
// runs on a background goroutine; the webview owns the main thread (required).
func nativeApp() error {
	url, errc, err := startServer()
	if err != nil {
		return err
	}
	fmt.Printf("cascade app (native webview) » %s\n", url)

	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("cascade")
	w.SetSize(1440, 900, webview.HintNone)
	w.Navigate(url)
	w.Run() // blocks until the window is closed

	select {
	case e := <-errc:
		return e
	default:
		return nil
	}
}
