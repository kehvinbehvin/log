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
	Consume(chan []rune) (chan []rune, chan []int, error)
}

func RemoveAlphanumeric(input []rune) []rune {
	content := make([]rune, len(input))
	copy(content, input)

	for i, r := range content {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			content[i] = topLevelAlphaNumericContent
		}
	}

	return content
}

func Compress(input []rune, rawInput []rune) ([]rune, [][]rune, error) {
	var counter int
	contentCounter := 0
	content := make([]rune, len(input))
	copy(content, input)
	tokens := make([][]rune, 0)

	for i, current := range input {
		if current == topLevelAlphaNumericContent {
			contentCounter++
		}

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
			tokens = append(tokens, input[i+1-contentCounter:i+1])
			contentCounter = 0
		}
	}

	return content[:counter], tokens, nil
}

func Maskify(input []rune, closingSym rune) ([]rune, int, [][]rune, error) {
	var content []rune
	var compressedContent [][]rune
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
				return []rune{}, 0, [][]rune{}, err
			}

			// Fast forward + offset
			//fmt.Println(string(input[i+1 : i+depth+1]))
			compressedContent = append(compressedContent, input[i+1:i+depth+1])
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

func (mc *MaskConsumer) Mask(input []rune) ([]rune, [][]rune, error) {
	maskedSymbols, _, tokens, err := Maskify(input, 0)
	if err != nil {
		return []rune{}, [][]rune{}, err
	}

	compressed, _, err := Compress(maskedSymbols, input)
	fmt.Println(string(compressed))
	if err != nil {
		return []rune{}, [][]rune{}, err
	}

	return compressed, tokens, nil
}

func (mc *MaskConsumer) Consume(in chan []rune) (chan []rune, chan [][]rune, error) {
	out := make(chan []rune, 100)
	tokenOut := make(chan [][]rune, 100)

	go func() {
		defer close(out)
		defer close(tokenOut)

		for log := range in {
			mask, tokens, err := mc.Mask(log)
			if err != nil {
				fmt.Println("consumer error with masking")
			}

			out <- mask
			tokenOut <- tokens
		}
	}()

	return out, tokenOut, nil
}
