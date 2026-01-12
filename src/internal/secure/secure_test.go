package secure

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestHashes(t *testing.T) {
	tests := []struct {
		hashResolver HashResolver
		password     string
		hash         string
		shouldMatch  bool
	}{
		{hashResolver: &BcryptResolver{}, password: "Hello, World!", hash: "$2a$10$k8fBIpJInrE70BzYy5rO/OUSt1w2.IX0bWhiMdb2mJEhjheVHDhvK", shouldMatch: true},
		{hashResolver: &BcryptResolver{}, password: "Hello, World!", hash: "$2a$10$X8fBIpJonrE70BzYy5rO/OUSt1w2.IX0bWhiMdb2mJEhjheVHDhvK", shouldMatch: false},
		{hashResolver: &BcryptResolver{}, password: "Hello, World!", hash: "", shouldMatch: false},
		{hashResolver: &BcryptResolver{}, password: "", hash: "", shouldMatch: false},
		{hashResolver: &MD5Resolver{}, password: "Hello, World!", hash: "65a8e27d8879283831b664bd8b7f0ad4", shouldMatch: true},
		{hashResolver: &MD5Resolver{}, password: "Hello, World!", hash: "X5a8e27d8879283831b664bd8b7f0ad4", shouldMatch: false},
		{hashResolver: &MD5Resolver{}, password: "Hello, World!", hash: "", shouldMatch: false},
		{hashResolver: &MD5Resolver{}, password: "", hash: "", shouldMatch: false},
		{hashResolver: &PlaintextResolver{}, password: "Hello, World!", hash: "Hello, World!", shouldMatch: true},
		{hashResolver: &PlaintextResolver{}, password: "Hello, World!", hash: "Xello, World!", shouldMatch: false},
		{hashResolver: &PlaintextResolver{}, password: "Hello, World!", hash: "", shouldMatch: false},
		{hashResolver: &PlaintextResolver{}, password: "", hash: "", shouldMatch: true},
		{hashResolver: &SHA512Resolver{}, password: "Hello, World!", hash: "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387", shouldMatch: true},
		{hashResolver: &SHA512Resolver{}, password: "Hello, World!", hash: "X74d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387", shouldMatch: false},
		{hashResolver: &SHA512Resolver{}, password: "Hello, World!", hash: "", shouldMatch: false},
		{hashResolver: &SHA512Resolver{}, password: "", hash: "", shouldMatch: false},
	}
	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			b := tt.hashResolver
			testza.AssertEqual(t, b.Check(tt.hash, tt.password), tt.shouldMatch, "Test: %#v", tt)
		})
	}
}

func TestGetMD5Hash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Hello World", "Hello, World!", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"Empty string", "", "d41d8cd98f00b204e9800998ecf8427e"},
		{"Single character", "a", "0cc175b9c0f1b6a831c399e269772661"},
		{"Numbers", "12345", "827ccb0eea8a706c4c34a16891f84e7b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMD5Hash(tt.input)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestGetMD5Hash_VariousInputs(t *testing.T) {
	tests := []string{
		"!@#$%",
		"测试",
		"This is a test",
		"123",
		"",
		"a",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			result := GetMD5Hash(input)
			// MD5 produces 32 hex characters (128 bits)
			testza.AssertEqual(t, 32, len(result), "MD5 hash should be 32 characters long")
			// Verify it's a valid hex string
			for _, char := range result {
				testza.AssertTrue(t, (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f'),
					"Hash should contain only hex characters")
			}
		})
	}
}

func TestGetMD5Hash_Consistency(t *testing.T) {
	input := "test input"
	hash1 := GetMD5Hash(input)
	hash2 := GetMD5Hash(input)
	testza.AssertEqual(t, hash1, hash2, "MD5 hash should be consistent")
}

func TestGetSHA512Hash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Hello World", "Hello, World!", "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387"},
		{"Empty string", "", "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSHA512Hash(tt.input)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestGetSHA512Hash_VariousInputs(t *testing.T) {
	tests := []string{
		"a",
		"12345",
		"!@#$%",
		"测试",
		"This is a test",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			result := GetSHA512Hash(input)
			// SHA512 produces 128 hex characters (512 bits)
			testza.AssertEqual(t, 128, len(result), "SHA512 hash should be 128 characters long")
			// Verify it's a valid hex string
			for _, char := range result {
				testza.AssertTrue(t, (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f'),
					"Hash should contain only hex characters")
			}
		})
	}
}

func TestGetSHA512Hash_Consistency(t *testing.T) {
	input := "test input"
	hash1 := GetSHA512Hash(input)
	hash2 := GetSHA512Hash(input)
	testza.AssertEqual(t, hash1, hash2, "SHA512 hash should be consistent")
}
