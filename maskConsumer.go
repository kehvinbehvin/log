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
	Consume(chan []rune) (chan []rune, error)
}

func RemoveAlphanumeric(input []rune) []rune {
	for i, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			input[i] = topLevelAlphaNumericContent
		}
	}

	return input
}

func Compress(input []rune) ([]rune, error) {
	var counter int

	for i, current := range input {
		// Append all symbols
		if current != topLevelAlphaNumericContent {
			input[counter] = current
			counter++
			continue
		}

		// Only append the last Y of connected Ys
		// Only append the last Y of the input
		if (i+1) == len(input) || input[i+1] != topLevelAlphaNumericContent {
			input[counter] = current
			counter++
		}
	}

	return input[:counter], nil
}

func Maskify(input []rune, closingSym rune) ([]rune, int, error) {
	var content []rune

	for i := 0; i < len(input); i++ {
		content = append(content, input[i])
		close, open := enclosingSymbols[input[i]]
		
		// Check for closing syms first because some closing symbols can be the same as their opening
		if input[i] == closingSym {
			// Closing Sym found, all nested content in this stack should be masked
			return []rune{nestedContent, closingSym}, i, nil 
		}
		
		if open {
			// State A: Closing sym found -> Mask returned
			// State B: Closing sym found but no content wanted -> empty rune slice returned
			innerContent, depth, err := Maskify(input[i + 1:], close)
			if err != nil {
				return []rune{}, 0, err
			}

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
	return content, len(input), nil
}

type MaskConsumer struct {}

func NewMaskConsumer() (*MaskConsumer) {
	return &MaskConsumer{}
}

func (mc *MaskConsumer) Mask(input []rune) ([]rune, error) {
	symbolsOnly := RemoveAlphanumeric(input)	
	maskedSymbols, _, err := Maskify(symbolsOnly, 0)
	if err != nil {
		return []rune{}, err
	}

	compressed, err := Compress(maskedSymbols)
	if err != nil {
		return []rune{}, err
	}

	return compressed, nil
}

func (mc *MaskConsumer) Consume(in chan []rune) (chan[]rune, error) {
	out := make(chan []rune, 100)
	
	go func() {
		defer close(out)

		for log := range in {
			mask, err := mc.Mask(log)
			if err != nil {
				fmt.Println("consumer error with masking")
			}
	
			out <- mask
		}
	}()

	return out, nil
}