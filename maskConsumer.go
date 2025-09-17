package main

import (
	"fmt"
)

const (
	nestedContent               = 'X'
	topLevelAlphaNumericContent = 'Y'
)

var enclosingSymbols = map[rune]rune{
	'[':  ']',
	'{':  '}',
	'<':  '>',
	'(':  ')',
	'"':  '"',
	'\'': '\'',
}

type Consumer interface {
	Consume(chan []rune) (chan Sentence, error)
}

func Compress(input []rune, rawInput []rune) (LogMask, error) {
	var counter int
	content := make([]rune, len(input))
	copy(content, input)

	for i, current := range input {
		// Append all symbols
		if current != topLevelAlphaNumericContent {
			content[counter] = current
			counter++
			continue
		}

		// Only append the last Y of connected Ys
		// Only append the last Y of the input
		if (i+1) == len(input) || input[i+1] != topLevelAlphaNumericContent {
			content[counter] = current
			counter++
		}
	}

	return content[:counter], nil
}

func Maskify(input []rune, closingSym rune) ([]rune, int, []Token, error) {
	var content []rune
	var compressedContent []Token
	var compressedContentCounter int

	for i := 0; i < len(input); i++ {
		r := input[i]
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			content = append(content, topLevelAlphaNumericContent)
			compressedContentCounter++
		} else {
			content = append(content, input[i])
			if compressedContentCounter > 0 {
				compressedContent = append(compressedContent, input[i-compressedContentCounter:i])
			}

			compressedContentCounter = 0
		}

		closing, opening := enclosingSymbols[input[i]]

		// Check for closing syms first because some closing symbols can be the same as their opening
		if input[i] == closingSym {
			// Closing Sym found, all nested content in this stack should be masked
			return []rune{nestedContent, closingSym}, i, compressedContent, nil
		}

		if opening {
			// State A: Closing sym found -> Mask returned
			// State B: Closing sym found but no content wanted -> empty rune slice returned
			innerContent, depth, _, err := Maskify(input[i+1:], closing)
			if err != nil {
				return []rune{}, 0, []Token{}, err
			}

			// Add raw content that will be compressed
			compressedContent = append(compressedContent, input[i+1:i+depth+1])
			// Fast forward + offset
			i = i + depth + 1

			// Append whatever Mask returns
			content = append(content, innerContent...)
			if i >= len(input) {
				break
			}

			continue
		}
	}

	// No closing symbols found, return whatever we have processed.
	// Amount processed is not len(content)
	return content, len(input), compressedContent, nil
}

type MaskConsumer struct{}

func NewMaskConsumer() *MaskConsumer {
	return &MaskConsumer{}
}

func (mc *MaskConsumer) Mask(input []rune) (Sentence, error) {
	maskedSymbols, _, tokens, err := Maskify(input, 0)
	if err != nil {
		return Sentence{}, err
	}

	compressed, err := Compress(maskedSymbols, input)
	if err != nil {
		return Sentence{}, err
	}

	return Sentence{
		Tokens: tokens,
		Mask:   compressed,
		Line:   input,
	}, nil
}

func (mc *MaskConsumer) Consume(in chan []rune) (chan Sentence, error) {
	sentenceChan := make(chan Sentence, 100)

	go func() {
		defer close(sentenceChan)

		for log := range in {
			sentence, err := mc.Mask(log)
			if err != nil {
				fmt.Println("consumer error with masking")
			}

			sentenceChan <- sentence
		}
	}()

	return sentenceChan, nil
}
