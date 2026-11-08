package i18n

import "testing"

func TestAuthenticationUICompleteLanguages(t *testing.T) {
	english := AuthenticationUI(LangEN)
	for _, lang := range []Language{LangEN, LangZH, LangFR, LangDE, LangIT, LangJA, LangKO} {
		got := AuthenticationUI(lang)
		if len(got) != len(english) {
			t.Fatalf("incomplete vocabulary: %s", lang)
		}
		for key := range english {
			if got[key] == "" {
				t.Errorf("missing %s in %s", key, lang)
			}
		}
	}
	copy := AuthenticationUI("unknown")
	if copy["signin"] != english["signin"] {
		t.Fatal("missing fallback")
	}
	copy["signin"] = "changed"
	if AuthenticationUI(LangEN)["signin"] == "changed" {
		t.Fatal("caller mutated shared vocabulary")
	}
}
