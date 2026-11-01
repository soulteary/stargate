package handlers

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// The browser toggles the show class after an unsuccessful login or send-code
// request. An inline display:none takes precedence and silently hides every
// error, even when the server correctly rejects the request.
func TestWardenLoginErrorFeedbackIsNotPermanentlyHidden(t *testing.T) {
	page, err := template.ParseFiles("../web/templates/login.warden.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []struct {
		name string
		data map[string]any
	}{
		{"email", map[string]any{"HeraldEnabled": true, "LoginEmailEnabled": true}},
		{"sms", map[string]any{"HeraldEnabled": true, "LoginSMSEnabled": true}},
		{"totp", map[string]any{"HeraldEnabled": true, "HeraldTOTPEnabled": true}},
		{"password", map[string]any{"PasswordLoginEnabled": true}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := page.Execute(&rendered, mode.data); err != nil {
				t.Fatal(err)
			}
			doc, err := html.Parse(&rendered)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			var visit func(*html.Node)
			visit = func(n *html.Node) {
				attrs := map[string]string{}
				for _, a := range n.Attr {
					attrs[a.Key] = a.Val
				}
				if attrs["id"] == "error-container" {
					found = true
					if _, hidden := attrs["hidden"]; hidden || strings.TrimSpace(attrs["style"]) != "" {
						t.Error("error feedback must be controlled by CSS classes, without an inline visibility override")
					}
					if attrs["role"] != "alert" || attrs["aria-atomic"] != "true" {
						t.Error("login errors must be announced as one accessible alert")
					}
				}
				for child := n.FirstChild; child != nil; child = child.NextSibling {
					visit(child)
				}
			}
			visit(doc)
			if !found {
				t.Fatal("rendered login page is missing its error feedback container")
			}
		})
	}
}
