package main

import (
	"errors"
	"fmt"
)

const (
	pct rune = '%'

	verbBool   rune = 't'
	verbInt    rune = 'd'
	verbString rune = 's'
	// TODO: Add missing verbs.
)

var (
	// ErrBadArg reports a bad argument.
	ErrBadArg = errors.New("bad argument")

	// ErrNoMatch reports that 'str' does not match 'format'.
	ErrNoMatch = errors.New("'str' does not match 'format'")

	// ErrMultipleMatches reports that 'str' matches 'format' more than once.
	ErrMultipleMatches = errors.New("'str' matches 'format' more than once")

	// ErrBug reports a bug.
	ErrBug = errors.New("bug")
)

// TODO: Initialize exported pattern type safe for (concurrent) reuse. Must compile equivalent.

// Gimmef uses 'format' to capture typed values from 'str' and assign them to 'targetPtrs'.
func Gimmef(format, str string, targetPtrs ...interface{}) error {
	if format == "" {
		return fmt.Errorf("%w: 'format' must not be empty", ErrBadArg)
	}

	if str == "" {
		return fmt.Errorf("%w: 'str' must not be empty", ErrBadArg)
	}

	if len(targetPtrs) == 0 {
		return fmt.Errorf("%w: one or more 'targetPtrs' required", ErrBadArg)
	}

	pattern, err := newPattern(format)
	if err != nil {
		return fmt.Errorf("parsing 'format': %w", err)
	}

	if len(targetPtrs) != pattern.verbCount() {
		return fmt.Errorf("found %d verbs for %d 'targetPtrs'; count must match", pattern.verbCount(), len(targetPtrs))
	}

	err = pattern.capture(str)
	if err != nil {
		return fmt.Errorf("applying 'format' to 'str': %w", err)
	}

	err = pattern.assign(targetPtrs)
	if err != nil {
		return fmt.Errorf("assigning values to 'targetPtrs': %w", err)
	}

	return nil
}
