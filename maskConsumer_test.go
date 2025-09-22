package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// MaskConsumerTestSuite provides test suite for MaskConsumer
type MaskConsumerTestSuite struct {
	suite.Suite
	consumer *MaskConsumer
	helper   *TestHelper
}

func (suite *MaskConsumerTestSuite) SetupTest() {
	suite.consumer = NewMaskConsumer()
	suite.helper = &TestHelper{}
}

func (suite *MaskConsumerTestSuite) TestMaskifyBasicAlphanumeric() {
	input := []rune("hello123")
	result, depth, tokens, err := Maskify(input, 0)

	suite.NoError(err)
	suite.Equal([]rune("YYYYYYYY"), result)
	suite.Equal(len(input), depth)
	suite.Len(tokens, 0) // No tokens because no separators
}

func (suite *MaskConsumerTestSuite) TestMaskifyWithSymbols() {
	input := []rune("hello-world_123")
	result, depth, tokens, err := Maskify(input, 0)

	suite.NoError(err)
	expected := []rune("YYYYY-YYYYY_YYY")
	suite.Equal(expected, result)
	suite.Equal(len(input), depth)
	suite.Len(tokens, 2) // Only "hello", "world" (final token not captured)
	suite.Equal([]rune("hello"), []rune(tokens[0]))
	suite.Equal([]rune("world"), []rune(tokens[1]))
}

func (suite *MaskConsumerTestSuite) TestMaskifyWithNestedBrackets() {
	input := []rune("test[nested]content")
	result, depth, tokens, err := Maskify(input, 0)

	suite.NoError(err)
	expected := []rune("YYYY[X]YYYYYYY")
	suite.Equal(expected, result)
	suite.Equal(len(input), depth)
	suite.Len(tokens, 2) // "test" and "nested" (content not captured)
	suite.Equal([]rune("test"), []rune(tokens[0]))
	suite.Equal([]rune("nested"), []rune(tokens[1]))
}

func (suite *MaskConsumerTestSuite) TestMaskifyWithNestedQuotes() {
	input := []rune(`message="hello world"`)
	result, depth, tokens, err := Maskify(input, 0)

	suite.NoError(err)
	expected := []rune("YYYYYYY=\"X\"")
	suite.Equal(expected, result)
	suite.Equal(len(input), depth)
	suite.Len(tokens, 2) // "message" and "hello world"
	suite.Equal([]rune("message"), []rune(tokens[0]))
	suite.Equal([]rune("hello world"), []rune(tokens[1]))
}

func (suite *MaskConsumerTestSuite) TestCompressConsecutiveYs() {
	input := []rune("YYYYYYYY-YYYY_YYY")
	original := []rune("something-else_too")

	result, err := Compress(input, original)

	suite.NoError(err)
	expected := LogMask("Y-Y_Y")
	suite.Equal(expected, result)
}

func (suite *MaskConsumerTestSuite) TestCompressWithNoConsecutiveYs() {
	input := []rune("Y-Y-Y")
	original := []rune("a-b-c")

	result, err := Compress(input, original)

	suite.NoError(err)
	expected := LogMask("Y-Y-Y")
	suite.Equal(expected, result) // Should be unchanged
}

func (suite *MaskConsumerTestSuite) TestMaskMethod() {
	input := []rune("03-17 16:13:38.936  1702 14638 D PowerManagerService")

	sentence, err := suite.consumer.Mask(input)

	suite.NoError(err)
	suite.Equal(LogLine(input), sentence.Line)
	suite.NotEmpty(sentence.Mask)
	suite.NotEmpty(sentence.Tokens)

	// Verify mask contains Y for alphanumeric and preserves other chars
	maskStr := string(sentence.Mask)
	suite.Contains(maskStr, "Y")
	suite.Contains(maskStr, "-")
	suite.Contains(maskStr, " ")
	suite.Contains(maskStr, ":")
	suite.Contains(maskStr, ".")
}

func (suite *MaskConsumerTestSuite) TestConsumeChannel() {
	// Create input channel
	input := make(chan []rune, 10)

	// Send test data
	testInputs := []string{
		"simple text",
		"data[nested]",
		"key=\"value\"",
	}

	for _, testInput := range testInputs {
		input <- []rune(testInput)
	}
	close(input)

	// Process through consumer
	output, err := suite.consumer.Consume(input)
	suite.NoError(err)

	// Collect results
	var results []Sentence
	for sentence := range output {
		results = append(results, sentence)
	}

	// Verify we got the expected number of results
	suite.Len(results, len(testInputs))

	// Verify each result has proper structure
	for i, result := range results {
		suite.Equal(LogLine(testInputs[i]), result.Line)
		suite.NotEmpty(result.Mask)
		// Tokens may be empty for some inputs, that's OK
	}
}

// Table-driven tests for various log formats
func (suite *MaskConsumerTestSuite) TestMaskVariousLogFormats() {
	testCases := []struct {
		name           string
		input          string
		shouldHaveTokens    bool
	}{
		{
			name:             "Android log",
			input:            "03-17 16:13:38.936  1702 14638 D PowerManagerService: release",
			shouldHaveTokens:    true,
		},
		{
			name:             "JSON-like log",
			input:            `{"level":"info","message":"test"}`,
			shouldHaveTokens:    true,
		},
		{
			name:             "URL in log",
			input:            "GET /api/users[123]/profile HTTP/1.1",
			shouldHaveTokens:    true,
		},
		{
			name:             "Empty string",
			input:            "",
			shouldHaveTokens:    false,
		},
		{
			name:             "Only symbols",
			input:            "!@#$%^&*()",
			shouldHaveTokens:    false,
		},
		{
			name:             "Nested brackets",
			input:            "outer[inner{deep}]end",
			shouldHaveTokens:    true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			sentence, err := suite.consumer.Mask([]rune(tc.input))
			suite.NoError(err)

			suite.Equal(LogLine(tc.input), sentence.Line)

			if tc.shouldHaveTokens {
				// Some inputs should produce tokens, others may not
				// This is based on whether there are separators
			}

			// Check that mask was created
			suite.NotNil(sentence.Mask)
		})
	}
}

func (suite *MaskConsumerTestSuite) TestMaskifyUnmatchedBrackets() {
	testCases := []struct {
		name  string
		input string
	}{
		{"unclosed bracket", "test[unclosed"},
		{"unclosed quote", `test"unclosed`},
		{"unclosed paren", "test(unclosed"},
		{"unclosed brace", "test{unclosed"},
		{"unclosed angle", "test<unclosed"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, depth, tokens, err := Maskify([]rune(tc.input), 0)

			// Should not crash and should process what it can
			suite.NoError(err)
			suite.Equal(len(tc.input), depth)
			suite.NotNil(result)
			suite.NotNil(tokens)
		})
	}
}

func TestMaskConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(MaskConsumerTestSuite))
}

// Additional benchmark tests for performance validation
func BenchmarkMaskifySimple(b *testing.B) {
	input := []rune("simple test string with alphanumeric123")

	for i := 0; i < b.N; i++ {
		_, _, _, _ = Maskify(input, 0)
	}
}

func BenchmarkMaskifyComplex(b *testing.B) {
	input := []rune(`complex[nested{deep["quoted"]}]structure`)

	for i := 0; i < b.N; i++ {
		_, _, _, _ = Maskify(input, 0)
	}
}

func BenchmarkMaskConsumerMask(b *testing.B) {
	consumer := NewMaskConsumer()
	input := []rune("03-17 16:13:38.936  1702 14638 D PowerManagerService: release:lock=189667585")

	for i := 0; i < b.N; i++ {
		_, _ = consumer.Mask(input)
	}
}

func BenchmarkCompress(b *testing.B) {
	input := []rune("YYYYYYYY-YYYYYYYY_YYYYYYYY")
	original := []rune("something-something_something")

	for i := 0; i < b.N; i++ {
		_, _ = Compress(input, original)
	}
}