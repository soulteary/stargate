// Package web contains the authentication resources shipped with the binary.
package web

import (
	"embed"
	"io/fs"
)

// Keep this list explicit: generated files and configuration must never be
// included by a wildcard when the resource directory grows.
//
//go:embed templates/login.html templates/login.warden.html templates/account.html templates/totp_start.html templates/totp_enroll.html templates/totp_revoke.html templates/assets/auth-ui.js templates/assets/auth-ui.css templates/assets/qrcode.min.js templates/assets/favicon.ico
var resources embed.FS

// Templates returns the immutable built-in template tree. Callers may continue
// to prefer an explicitly deployed external template directory.
func Templates() fs.FS { return sub("templates") }

// Assets exposes only static resources, never the template or configuration tree.
func Assets() fs.FS { return sub("templates/assets") }

func sub(directory string) fs.FS {
	result, err := fs.Sub(resources, directory)
	if err != nil {
		panic("invalid built-in resource directory")
	} // constant valid paths, checked at build time
	return result
}
