package config

import "testing"

func Test_HasHTTPScheme_DetectsInsecureURLs(t *testing.T) {
	cases := map[string]bool{
		"http://idp.local/auth":  true,
		"https://idp.local/auth": false,
		"idp.local/auth":         false,
		"":                       false,
	}

	for rawURL, expected := range cases {
		if hasHTTPScheme(rawURL) != expected {
			t.Errorf("hasHTTPScheme(%q) = %v, want %v", rawURL, hasHTTPScheme(rawURL), expected)
		}
	}
}
