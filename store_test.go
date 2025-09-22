package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// MemoryStoreTestSuite provides test suite for MemoryStore
type MemoryStoreTestSuite struct {
	suite.Suite
	store   *MemoryStore[string]
	boolStore *MemoryStore[bool]
	helper  *TestHelper
}

func (suite *MemoryStoreTestSuite) SetupTest() {
	suite.store = &MemoryStore[string]{
		data: make(map[string]string),
	}
	suite.boolStore = NewMemoryStore()
	suite.helper = &TestHelper{}
}

func (suite *MemoryStoreTestSuite) TestPutAndGet() {
	// Test putting and getting a value
	err := suite.store.Put("test-key", "test-value")
	suite.NoError(err)

	value, err := suite.store.Get("test-key")
	suite.NoError(err)
	suite.Equal("test-value", value)
}

func (suite *MemoryStoreTestSuite) TestGetNonExistentKey() {
	// Test getting a non-existent key returns error
	value, err := suite.store.Get("non-existent")
	suite.Error(err)
	suite.Equal("", value) // Zero value for string
	suite.Contains(err.Error(), "key not found")
}

func (suite *MemoryStoreTestSuite) TestOverwriteValue() {
	// Test overwriting an existing value
	suite.store.Put("key", "original")
	suite.store.Put("key", "updated")

	value, err := suite.store.Get("key")
	suite.NoError(err)
	suite.Equal("updated", value)
}

func (suite *MemoryStoreTestSuite) TestBoolStoreCreation() {
	// Test NewMemoryStore creates proper bool store
	boolStore := NewMemoryStore()
	suite.NotNil(boolStore)
	suite.NotNil(boolStore.data)

	// Test putting and getting bool values
	err := boolStore.Put("test", true)
	suite.NoError(err)

	value, err := boolStore.Get("test")
	suite.NoError(err)
	suite.True(value)
}

func (suite *MemoryStoreTestSuite) TestContextStoreCreation() {
	// Test NewContextStore creates proper Context store
	contextStore := NewContextStore()
	suite.NotNil(contextStore)
	suite.NotNil(contextStore.data)

	// Test putting and getting Context values
	testContext := Context{labels: []string{"label1", "label2"}}
	err := contextStore.Put("test-mask", testContext)
	suite.NoError(err)

	value, err := contextStore.Get("test-mask")
	suite.NoError(err)
	suite.Equal(testContext.labels, value.labels)
}

func (suite *MemoryStoreTestSuite) TestReportToFile() {
	// Populate store with test data
	suite.store.Put("key1", "value1")
	suite.store.Put("key2", "value2")
	suite.store.Put("key3", "value3")

	// Create temp directory for test output
	tmpDir, err := os.MkdirTemp("", "store_test_*")
	suite.NoError(err)
	defer os.RemoveAll(tmpDir)

	reportFile := filepath.Join(tmpDir, "test_report.txt")

	// Test Report method
	err = suite.store.Report(reportFile)
	suite.NoError(err)

	// Verify file was created and contains expected content
	content, err := os.ReadFile(reportFile)
	suite.NoError(err)

	contentStr := string(content)
	suite.Contains(contentStr, "key1")
	suite.Contains(contentStr, "key2")
	suite.Contains(contentStr, "key3")

	// Verify each key is on its own line
	lines := strings.Split(strings.TrimSpace(contentStr), "\n")
	suite.Len(lines, 3)
}

func (suite *MemoryStoreTestSuite) TestReportFileCreationError() {
	// Test Report with invalid file path
	err := suite.store.Report("/invalid/path/that/does/not/exist/report.txt")
	suite.Error(err)
}

func (suite *MemoryStoreTestSuite) TestEmptyStoreReport() {
	// Test reporting empty store
	tmpDir, err := os.MkdirTemp("", "store_test_*")
	suite.NoError(err)
	defer os.RemoveAll(tmpDir)

	reportFile := filepath.Join(tmpDir, "empty_report.txt")

	err = suite.store.Report(reportFile)
	suite.NoError(err)

	// Verify empty file
	content, err := os.ReadFile(reportFile)
	suite.NoError(err)
	suite.Empty(content)
}

func (suite *MemoryStoreTestSuite) TestConcurrentAccess() {
	// Test concurrent put operations (basic test)
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			key := fmt.Sprintf("key-%d", index)
			value := fmt.Sprintf("value-%d", index)
			suite.store.Put(key, value)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all values were stored
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		expectedValue := fmt.Sprintf("value-%d", i)

		value, err := suite.store.Get(key)
		suite.NoError(err)
		suite.Equal(expectedValue, value)
	}
}

// Table-driven tests for edge cases
func (suite *MemoryStoreTestSuite) TestEdgeCases() {
	testCases := []struct {
		name     string
		key      string
		value    string
		expectError bool
	}{
		{"empty key", "", "value", false},
		{"empty value", "key", "", false},
		{"special chars key", "key!@#$%^&*()", "value", false},
		{"unicode key", "キー", "値", false},
		{"long key", strings.Repeat("a", 1000), "value", false},
		{"long value", "key", strings.Repeat("v", 1000), false},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := suite.store.Put(tc.key, tc.value)
			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)

				value, err := suite.store.Get(tc.key)
				suite.NoError(err)
				suite.Equal(tc.value, value)
			}
		})
	}
}

func TestMemoryStoreTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryStoreTestSuite))
}

// Additional standalone tests using assert package
func TestMemoryStoreStandalone(t *testing.T) {
	store := &MemoryStore[int]{
		data: make(map[string]int),
	}

	// Test integer type store
	assert.NoError(t, store.Put("number", 42))

	value, err := store.Get("number")
	assert.NoError(t, err)
	assert.Equal(t, 42, value)

	// Test zero value for int
	_, err = store.Get("missing")
	assert.Error(t, err)
}