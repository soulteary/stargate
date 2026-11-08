package config

import "testing"

func TestDingtalkUserIDRequirementValidation(t *testing.T) {
	old := LoginDingtalkRequireUserID.Value
	t.Cleanup(func() { LoginDingtalkRequireUserID.Value = old })
	for _, tc := range []struct {
		name, value        string
		set, want, invalid bool
	}{
		{name: "default false"}, {name: "explicit false", value: "false", set: true}, {name: "true", value: "true", set: true, want: true},
		{name: "case insensitive", value: "TRUE", set: true, want: true}, {name: "bad bool", value: "1", set: true, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv("LOGIN_DINGTALK_REQUIRE_USERID", tc.value)
			} else {
				t.Setenv("LOGIN_DINGTALK_REQUIRE_USERID", "")
			}
			err := LoginDingtalkRequireUserID.Validate()
			if (err != nil) != tc.invalid {
				t.Fatalf("validation=%v invalid=%v", err, tc.invalid)
			}
			if err == nil && LoginDingtalkRequireUserID.ToBool() != tc.want {
				t.Fatal("unexpected bool")
			}
		})
	}
}
