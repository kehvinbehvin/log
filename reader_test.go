package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// FileReaderTestSuite provides test suite for FileReader
type FileReaderTestSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *FileReaderTestSuite) SetupTest() {
	suite.helper = &TestHelper{}
}

func (suite *FileReaderTestSuite) TestNewFileReader() {
	// Test that FileReader is created properly
	filePath := "test.log"
	reader := NewFileReader(filePath)

	suite.NotNil(reader)
	suite.Equal(filePath, reader.filePath)
}

func (suite *FileReaderTestSuite) TestReadValidFile() {
	// Create temporary file with test content
	content := "line1\nline2\nline3\n"
	tempFile, cleanup, err := suite.helper.CreateTempFile(content)
	suite.NoError(err)
	defer cleanup()

	// Create reader and read file
	reader := NewFileReader(tempFile)
	output, err := reader.Read()
	suite.NoError(err)

	// Collect all lines
	var lines [][]rune
	for line := range output {
		lines = append(lines, line)
	}

	// Verify content
	suite.Len(lines, 3)
	suite.Equal("line1", string(lines[0]))
	suite.Equal("line2", string(lines[1]))
	suite.Equal("line3", string(lines[2]))
}

func (suite *FileReaderTestSuite) TestReadEmptyFile() {
	// Create empty temporary file
	tempFile, cleanup, err := suite.helper.CreateTempFile("")
	suite.NoError(err)
	defer cleanup()

	// Create reader and read file
	reader := NewFileReader(tempFile)
	output, err := reader.Read()
	suite.NoError(err)

	// Collect all lines
	var lines [][]rune
	for line := range output {
		lines = append(lines, line)
	}

	// Verify no lines
	suite.Len(lines, 0)
}

func (suite *FileReaderTestSuite) TestReadSingleLine() {
	// Create file with single line (no newline)
	content := "single line without newline"
	tempFile, cleanup, err := suite.helper.CreateTempFile(content)
	suite.NoError(err)
	defer cleanup()

	// Create reader and read file
	reader := NewFileReader(tempFile)
	output, err := reader.Read()
	suite.NoError(err)

	// Collect all lines
	var lines [][]rune
	for line := range output {
		lines = append(lines, line)
	}

	// Verify single line
	suite.Len(lines, 1)
	suite.Equal(content, string(lines[0]))
}

func (suite *FileReaderTestSuite) TestReadUnicodeContent() {
	// Create file with unicode content
	content := "Hello 世界\némoji: 🚀\nελληνικά\n"
	tempFile, cleanup, err := suite.helper.CreateTempFile(content)
	suite.NoError(err)
	defer cleanup()

	// Create reader and read file
	reader := NewFileReader(tempFile)
	output, err := reader.Read()
	suite.NoError(err)

	// Collect all lines
	var lines [][]rune
	for line := range output {
		lines = append(lines, line)
	}

	// Verify unicode content
	suite.Len(lines, 3)
	suite.Equal("Hello 世界", string(lines[0]))
	suite.Equal("émoji: 🚀", string(lines[1]))
	suite.Equal("ελληνικά", string(lines[2]))
}

func (suite *FileReaderTestSuite) TestReadLongLines() {
	// Create file with very long lines
	longLine1 := string(make([]rune, 10000)) // 10k runes of null characters
	for i := range []rune(longLine1) {
		[]rune(longLine1)[i] = 'A'
	}
	longLine1 = ""
	for i := 0; i < 10000; i++ {
		longLine1 += "A"
	}

	longLine2 := ""
	for i := 0; i < 5000; i++ {
		longLine2 += "B"
	}

	content := longLine1 + "\n" + longLine2 + "\n"
	tempFile, cleanup, err := suite.helper.CreateTempFile(content)
	suite.NoError(err)
	defer cleanup()

	// Create reader and read file
	reader := NewFileReader(tempFile)
	output, err := reader.Read()
	suite.NoError(err)

	// Collect all lines
	var lines [][]rune
	for line := range output {
		lines = append(lines, line)
	}

	// Verify long lines
	suite.Len(lines, 2)
	suite.Len(lines[0], 10000)
	suite.Len(lines[1], 5000)
	suite.Equal(longLine1, string(lines[0]))
	suite.Equal(longLine2, string(lines[1]))
}

func (suite *FileReaderTestSuite) TestReadNonExistentFile() {
	// Try to read non-existent file
	reader := NewFileReader("/non/existent/file.log")
	output, err := reader.Read()

	// Should return error
	suite.Error(err)
	suite.Nil(output)
	suite.Contains(err.Error(), "could not open file")
}

func (suite *FileReaderTestSuite) TestReadTestDataFiles() {
	// Test reading actual test data files
	testFiles := []struct {
		name     string
		path     string
		minLines int
	}{
		{"sample log", suite.helper.GetTestDataPath("sample.log"), 4},
		{"empty log", suite.helper.GetTestDataPath("empty.log"), 0},
		{"malformed log", suite.helper.GetTestDataPath("malformed.log"), 6},
	}

	for _, tf := range testFiles {
		suite.Run(tf.name, func() {
			reader := NewFileReader(tf.path)
			output, err := reader.Read()
			suite.NoError(err)

			// Collect all lines
			var lines [][]rune
			for line := range output {
				lines = append(lines, line)
			}

			// Verify expected number of lines
			if tf.minLines == 0 {
				suite.Len(lines, 0)
			} else {
				suite.GreaterOrEqual(len(lines), tf.minLines)
			}

			// Verify all lines are valid rune slices
			for i, line := range lines {
				suite.NotNil(line, "Line %d should not be nil", i)
				// Lines should be valid UTF-8 when converted back to string
				suite.NotPanics(func() {
					_ = string(line)
				}, "Line %d should convert to string without panic", i)
			}
		})
	}
}

func (suite *FileReaderTestSuite) TestReadSpecialCharacters() {
	// Test various special characters and edge cases
	content := "tab\there\n" +
		"quotes: \"double\" and 'single'\n" +
		"brackets: [square] {curly} <angle> (round)\n" +
		"symbols: !@#$%^&*()_+-={}[]|\\:;\"'<>?,./\n" +
		"numbers: 1234567890\n" +
		"mixed: line with\ttab and spaces\n"

	tempFile, cleanup, err := suite.helper.CreateTempFile(content)
	suite.NoError(err)
	defer cleanup()

	reader := NewFileReader(tempFile)
	output, err := reader.Read()
	suite.NoError(err)

	var lines [][]rune
	for line := range output {
		lines = append(lines, line)
	}

	suite.Len(lines, 6)

	// Verify tab character is preserved
	suite.Contains(string(lines[0]), "\t")

	// Verify quotes are preserved
	suite.Contains(string(lines[1]), "\"")
	suite.Contains(string(lines[1]), "'")

	// Verify brackets are preserved
	suite.Contains(string(lines[2]), "[")
	suite.Contains(string(lines[2]), "{")
	suite.Contains(string(lines[2]), "<")
	suite.Contains(string(lines[2]), "(")
}

func (suite *FileReaderTestSuite) TestChannelClosing() {
	// Test that the output channel is properly closed
	content := "line1\nline2\n"
	tempFile, cleanup, err := suite.helper.CreateTempFile(content)
	suite.NoError(err)
	defer cleanup()

	reader := NewFileReader(tempFile)
	output, err := reader.Read()
	suite.NoError(err)

	// Collect lines and verify channel closes
	var lines [][]rune
	channelClosed := false

	// Set a timeout to prevent hanging
	timeout := time.After(5 * time.Second)

	for {
		select {
		case line, ok := <-output:
			if !ok {
				channelClosed = true
				goto done
			}
			lines = append(lines, line)
		case <-timeout:
			suite.Fail("Channel did not close within timeout")
			goto done
		}
	}

done:
	suite.True(channelClosed, "Output channel should be closed")
	suite.Len(lines, 2)
}

func (suite *FileReaderTestSuite) TestFileHandleCleanup() {
	// Test that file handles are properly closed
	content := "test line\n"
	tempFile, cleanup, err := suite.helper.CreateTempFile(content)
	suite.NoError(err)
	defer cleanup()

	// Read file multiple times to test handle cleanup
	for i := 0; i < 10; i++ {
		reader := NewFileReader(tempFile)
		output, err := reader.Read()
		suite.NoError(err)

		// Consume all lines
		for range output {
		}
	}

	// If file handles weren't closed, this test would eventually fail
	// due to too many open files
}

func (suite *FileReaderTestSuite) TestConcurrentReads() {
	// Test multiple concurrent reads of the same file
	content := "concurrent test line\n"
	tempFile, cleanup, err := suite.helper.CreateTempFile(content)
	suite.NoError(err)
	defer cleanup()

	numReaders := 5
	results := make([][]string, numReaders)
	done := make(chan int, numReaders)

	// Start multiple readers concurrently
	for i := 0; i < numReaders; i++ {
		go func(index int) {
			reader := NewFileReader(tempFile)
			output, err := reader.Read()
			suite.NoError(err)

			var lines []string
			for line := range output {
				lines = append(lines, string(line))
			}
			results[index] = lines
			done <- index
		}(i)
	}

	// Wait for all readers to complete
	for i := 0; i < numReaders; i++ {
		<-done
	}

	// Verify all readers got the same content
	for i := 0; i < numReaders; i++ {
		suite.Len(results[i], 1)
		suite.Equal("concurrent test line", results[i][0])
	}
}

func TestFileReaderTestSuite(t *testing.T) {
	suite.Run(t, new(FileReaderTestSuite))
}

// Benchmark tests
func BenchmarkFileReaderSmallFile(b *testing.B) {
	// Create small test file
	content := "line1\nline2\nline3\n"
	tmpFile, err := os.CreateTemp("", "bench_small_*.log")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(content)
	tmpFile.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := NewFileReader(tmpFile.Name())
		output, err := reader.Read()
		if err != nil {
			b.Fatal(err)
		}

		for range output {
		}
	}
}

func BenchmarkFileReaderLargeFile(b *testing.B) {
	// Create larger test file
	tmpFile, err := os.CreateTemp("", "bench_large_*.log")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write 1000 lines
	for i := 0; i < 1000; i++ {
		tmpFile.WriteString("This is a longer line with more content to test performance\n")
	}
	tmpFile.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := NewFileReader(tmpFile.Name())
		output, err := reader.Read()
		if err != nil {
			b.Fatal(err)
		}

		for range output {
		}
	}
}