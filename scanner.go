package main

import "fmt"

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
