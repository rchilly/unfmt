package unfmt

import (
	"errors"
	"fmt"
)

const (
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

	// ErrEmptyCapture reports a capture of empty string for one of the verbs in 'format'.
	ErrEmptyCapture = errors.New("captured empty string")

	// ErrBug reports a bug.
	ErrBug = errors.New("bug")
)

// Scanner stores information from a format string for the evaluation of multiple inputs against it.
type Scanner struct {
	p *pattern
}

// NewScanner initializes a Scanner from a format string.
func NewScanner(format string) (Scanner, error) {
	var s Scanner

	p, err := newPattern(format)
	if err != nil {
		return s, fmt.Errorf("initializing new scanner from 'format': %w", err)
	}

	s.p = &p

	return s, nil
}

// ScanString captures values from 'str' according to the Scanner's state and assigns them to 'targetPtrs'.
func (s Scanner) ScanString(str string, targetPtrs ...interface{}) error {
	if str == "" {
		return fmt.Errorf("%w: 'str' must not be empty", ErrBadArg)
	}

	if len(targetPtrs) != s.p.verbCount() {
		return fmt.Errorf("got %d 'targetPtrs' for %d verbs; count must match", len(targetPtrs), s.p.verbCount())
	}

	s.p.reset()

	err := s.p.capture(str)
	if err != nil {
		return fmt.Errorf("capturing from 'str': %w", err)
	}

	err = s.p.assign(targetPtrs)
	if err != nil {
		return fmt.Errorf("assigning values to 'targetPtrs': %w", err)
	}

	return nil
}

// TODO: Initialize exported pattern type safe for (concurrent) reuse. Must compile equivalent.

// ScanString captures values from 'str' according to 'format' and assigns them to 'targetPtrs'.
func ScanString(str, format string, targetPtrs ...interface{}) error {
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
		return fmt.Errorf("got %d 'targetPtrs' for %d verbs; count must match", len(targetPtrs), pattern.verbCount())
	}

	err = pattern.capture(str)
	if err != nil {
		return fmt.Errorf("capturing from 'str': %w", err)
	}

	err = pattern.assign(targetPtrs)
	if err != nil {
		return fmt.Errorf("assigning values to 'targetPtrs': %w", err)
	}

	return nil
}
