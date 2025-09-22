package main

import (
	"os"
	"path/filepath"
)

// TestHelper provides common utilities for tests
type TestHelper struct{}

// CreateTempFile creates a temporary file with given content for testing
func (th *TestHelper) CreateTempFile(content string) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "test_*.log")
	if err != nil {
		return "", nil, err
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, err
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", nil, err
	}

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup, nil
}

// GetTestDataPath returns the full path to a test data file
func (th *TestHelper) GetTestDataPath(filename string) string {
	return filepath.Join("testdata", filename)
}

// StringToRunes converts a string to []rune for testing
func (th *TestHelper) StringToRunes(s string) []rune {
	return []rune(s)
}

// RunesToString converts []rune back to string for assertions
func (th *TestHelper) RunesToString(r []rune) string {
	return string(r)
}

// CreateTestSentence creates a Sentence for testing purposes
func (th *TestHelper) CreateTestSentence(line string, tokens []string, mask string) Sentence {
	var runeTokens []Token
	for _, token := range tokens {
		runeTokens = append(runeTokens, []rune(token))
	}

	return Sentence{
		Tokens: runeTokens,
		Mask:   []rune(mask),
		Line:   []rune(line),
	}
}

// CreateTestContext creates a Context for testing purposes
func (th *TestHelper) CreateTestContext(labels []string) Context {
	return Context{labels: labels}
}