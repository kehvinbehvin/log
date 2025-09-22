package main

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// AdminTestSuite provides test suite for Admin
type AdminTestSuite struct {
	suite.Suite
	admin        *Admin
	maskStore    *MemoryStore[bool]
	contextStore *MemoryStore[Context]
	wg           *sync.WaitGroup
	helper       *TestHelper
}

func (suite *AdminTestSuite) SetupTest() {
	suite.maskStore = NewMemoryStore()
	suite.contextStore = NewContextStore()
	// Create a fresh WaitGroup for each test
	suite.wg = &sync.WaitGroup{}
	suite.admin = NewAdmin(suite.maskStore, suite.contextStore, suite.wg)
	suite.helper = &TestHelper{}
}

func (suite *AdminTestSuite) TestNewAdmin() {
	// Test that Admin is created properly
	suite.NotNil(suite.admin)
	suite.Equal(suite.maskStore, suite.admin.maskStore)
	suite.Equal(suite.contextStore, suite.admin.contextStore)
	suite.Equal(suite.wg, suite.admin.wg)
}

func (suite *AdminTestSuite) TestAdministrateUnregisteredMask() {
	// Create input channel with test sentence
	input := make(chan Sentence, 1)
	testSentence := suite.helper.CreateTestSentence(
		"test log line",
		[]string{"test", "log", "line"},
		"Y Y Y",
	)

	input <- testSentence
	close(input)

	suite.wg.Add(1) // Admin will call Done() once

	// Process through admin (mask is not pre-registered)
	unRegistered, registered, err := suite.admin.Administrate(input)
	suite.NoError(err)

	// Wait for processing to complete
	suite.wg.Wait()

	// Now collect from unregistered channel (should be closed by admin)
	var unregisteredSentences []Sentence
	for sentence := range unRegistered {
		unregisteredSentences = append(unregisteredSentences, sentence)
	}

	// Verify routing - should go to unregistered
	suite.Len(unregisteredSentences, 1)
	suite.Equal(testSentence, unregisteredSentences[0])

	// Verify mask was added to store as false (unregistered)
	maskKey := string(testSentence.Mask)
	status, err := suite.maskStore.Get(maskKey)
	suite.NoError(err)
	suite.False(status)

	// Check that registered channel has no data (non-blocking check)
	select {
	case <-registered:
		suite.Fail("Expected registered channel to be empty")
	case <-time.After(10 * time.Millisecond):
		// Good, channel is empty
	}
}

func (suite *AdminTestSuite) TestAdministrateRegisteredMask() {
	// Pre-register a mask as true (registered)
	testSentence := suite.helper.CreateTestSentence(
		"registered log line",
		[]string{"registered", "log"},
		"Y Y Y",
	)
	maskKey := string(testSentence.Mask)
	suite.maskStore.Put(maskKey, true)

	// Create input channel
	input := make(chan Sentence, 1)
	input <- testSentence
	close(input)

	suite.wg.Add(1) // Admin will call Done() once

	// Process through admin
	unRegistered, registered, err := suite.admin.Administrate(input)
	suite.NoError(err)

	// Wait for processing to complete
	suite.wg.Wait()

	// Collect from unregistered channel (should be closed and empty)
	var unregisteredSentences []Sentence
	for sentence := range unRegistered {
		unregisteredSentences = append(unregisteredSentences, sentence)
	}

	// Verify unregistered is empty
	suite.Len(unregisteredSentences, 0)

	// Check registered channel for data
	select {
	case sentence := <-registered:
		suite.Equal(testSentence, sentence)
	case <-time.After(100 * time.Millisecond):
		suite.Fail("Expected sentence in registered channel")
	}
}

func (suite *AdminTestSuite) TestAdministrateMultipleSentences() {
	// Create test sentences with different masks
	sentence1 := suite.helper.CreateTestSentence("line1", []string{"line1"}, "Y1")
	sentence2 := suite.helper.CreateTestSentence("line2", []string{"line2"}, "Y2")
	sentence3 := suite.helper.CreateTestSentence("line3", []string{"line3"}, "Y3")

	// Pre-register sentence2's mask
	suite.maskStore.Put(string(sentence2.Mask), true)

	// Create input channel
	input := make(chan Sentence, 3)
	input <- sentence1
	input <- sentence2
	input <- sentence3
	close(input)

	suite.wg.Add(1) // Admin will call Done() once

	// Process through admin
	unRegistered, registered, err := suite.admin.Administrate(input)
	suite.NoError(err)

	// Wait for processing to complete
	suite.wg.Wait()

	// Collect from unregistered channel
	var unregisteredSentences []Sentence
	for sentence := range unRegistered {
		unregisteredSentences = append(unregisteredSentences, sentence)
	}

	// Should have sentence1 and sentence3
	suite.Len(unregisteredSentences, 2)

	// Check registered channel for sentence2
	var foundSentence2 bool
	select {
	case sentence := <-registered:
		if sentence.Line[0] == 'l' && sentence.Line[4] == '2' { // "line2"
			foundSentence2 = true
		}
	case <-time.After(100 * time.Millisecond):
		// No sentence found
	}
	suite.True(foundSentence2, "Expected sentence2 in registered channel")
}

func (suite *AdminTestSuite) TestAdministrateEmptyInput() {
	// Create empty input channel
	input := make(chan Sentence)
	close(input)

	suite.wg.Add(1) // Admin will call Done() once

	// Process through admin
	unRegistered, registered, err := suite.admin.Administrate(input)
	suite.NoError(err)

	// Wait for processing to complete
	suite.wg.Wait()

	// Collect from unregistered channel (should be closed and empty)
	var unregisteredSentences []Sentence
	for sentence := range unRegistered {
		unregisteredSentences = append(unregisteredSentences, sentence)
	}

	// Verify no sentences were processed
	suite.Len(unregisteredSentences, 0)

	// Check registered channel is empty
	select {
	case <-registered:
		suite.Fail("Expected registered channel to be empty")
	case <-time.After(10 * time.Millisecond):
		// Good, channel is empty
	}
}

func (suite *AdminTestSuite) TestChannelTypes() {
	// Test that the returned channels are of correct types
	input := make(chan Sentence)
	close(input)

	suite.wg.Add(1) // Admin will call Done() once

	unRegistered, registered, err := suite.admin.Administrate(input)
	suite.NoError(err)

	// Verify channel types
	suite.IsType((UnRegisteredChan)(nil), unRegistered)
	suite.IsType((RegisteredChan)(nil), registered)

	suite.wg.Wait()
}

func (suite *AdminTestSuite) TestMaskStoreIntegration() {
	// Test that Admin properly interacts with mask store
	sentence := suite.helper.CreateTestSentence("test", []string{"test"}, "Y")
	maskKey := string(sentence.Mask)

	// Verify mask doesn't exist initially
	_, err := suite.maskStore.Get(maskKey)
	suite.Error(err) // Should error because key doesn't exist

	// Process sentence through admin
	input := make(chan Sentence, 1)
	input <- sentence
	close(input)

	suite.wg.Add(1)

	unRegistered, _, err := suite.admin.Administrate(input)
	suite.NoError(err)

	suite.wg.Wait()

	// Drain unregistered channel
	for range unRegistered {
	}

	// Verify mask was added to store as false
	status, err := suite.maskStore.Get(maskKey)
	suite.NoError(err)
	suite.False(status)
}

func TestAdminTestSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}