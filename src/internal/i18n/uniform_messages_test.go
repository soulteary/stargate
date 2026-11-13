package i18n

import "testing"

var allLanguages = []Language{LangEN, LangZH, LangFR, LangDE, LangIT, LangJA, LangKO}

func TestUniformAuthenticationUICompleteLanguages(t *testing.T) {
	english := UniformAuthenticationUI(LangEN)
	if len(english) != 3 || english["sent"] == "" || english["authentication_failed"] == "" || english["identifier_hint"] == "" {
		t.Fatalf("unexpected English phrases: %v", english)
	}
	for _, lang := range allLanguages {
		got, original := UniformAuthenticationUI(lang), AuthenticationUI(lang)
		if len(got) != len(english) {
			t.Fatalf("incomplete phrases: %s", lang)
		}
		for key := range english {
			if got[key] == "" {
				t.Errorf("missing %s in %s", key, lang)
			}
			if original[key] == "" {
				t.Errorf("%s does not replace an existing phrase in %s", key, lang)
			}
			if got[key] == original[key] {
				t.Errorf("%s is not replaced in %s", key, lang)
			}
			if lang != LangEN && got[key] == english[key] {
				t.Errorf("%s is not translated in %s", key, lang)
			}
		}
	}
	fallback := UniformAuthenticationUI("unknown")
	if fallback["sent"] != english["sent"] {
		t.Fatal("missing fallback")
	}
	fallback["sent"] = "changed"
	if UniformAuthenticationUI(LangEN)["sent"] == "changed" {
		t.Fatal("caller mutated shared phrases")
	}
}

func TestUniformServerMessagesCompleteLanguages(t *testing.T) {
	for _, key := range []string{"success.verify_code_sent_uniform", "error.login_failed_uniform"} {
		english := TWithLang(LangEN, key)
		if english == "" || english == key {
			t.Fatalf("missing English message for %s", key)
		}
		for _, lang := range allLanguages[1:] {
			got := TWithLang(lang, key)
			if got == "" || got == key || got == english {
				t.Errorf("%s is not translated in %s: %q", key, lang, got)
			}
		}
	}
}
