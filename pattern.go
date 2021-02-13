package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type pattern struct {
	format   string
	verbs    []substring
	segments []substring
	captures []string
}

type substring struct {
	value  string
	starts []int
}

func (s substring) String() string {
	return s.value
}

func newPattern(format string) (p pattern, err error) {
	err = p.parseVerbs(format)
	if err != nil {
		return
	}

	// After parsing verbs, must unescape '%%'s before parsing segments
	// in order to match literal '%'s in the string input.
	err = p.parseSegments(unescapeFormat(format))

	p.format = format
	return
}

func unescapeFormat(format string) string {
	return strings.ReplaceAll(format, fmt.Sprintf("%c%c", pct, pct), fmt.Sprintf("%c", pct))
}

func (p *pattern) parseVerbs(format string) error {
	fmtBytes := []byte(format)

	var seekVerb bool
	var idx int

	for len(fmtBytes) > 0 {
		nextRune, size := utf8.DecodeRune(fmtBytes)

		if seekVerb {
			if nextRune != pct {
				if !isSupportedVerb(nextRune) {
					return fmt.Errorf("%w: unsupported verb '%c%c'", ErrBadArg, pct, nextRune)
				}

				// Subtract from index to account for the preceding '%'.
				p.verbs = append(p.verbs, substring{
					starts: []int{idx - 1},
					value:  fmt.Sprintf("%c%c", pct, nextRune),
				})
			}

			seekVerb = false
		} else if nextRune == pct {
			seekVerb = true
		}

		idx += size

		fmtBytes = fmtBytes[size:]
	}

	return nil
}

/*
	Breaks a format string into the non-zero substrings in between
	each of its verbs and stores them on the pattern instance.

	The number of segments will vary, up to one more than
	the number of verbs, depending on the position of verbs
	in the format string. For example:

	Format 'my-%s-format-%s-string-%s-rocks' yields 4 segments
	'my-', '-format-', '-string-', and '-rocks' for 3 verbs.

	Format '%d + %d = %d' yields 2 segments ' + ' and ' = ',
	also for 3 verbs.

	Expects any literal '%'s in the format to be unescaped.
*/
func (p *pattern) parseSegments(unescapedFormat string) error {
	maxSegments := len(p.verbs) + 1
	p.segments = make([]substring, 0, maxSegments)

	remainder := unescapedFormat

	for i, verb := range p.verbs {
		halves := strings.SplitN(remainder, verb.value, 2)
		if len(halves[0]) > 0 {
			p.segments = append(p.segments, substring{
				value: halves[0],
			})
		} else if i > 0 {
			if verb.value == p.verbs[i-1].value {
				return fmt.Errorf("%w: found consecutive instances of verb '%s' without an intervening substring", ErrBadArg, verb.value)
			}
		}

		remainder = halves[1]
	}

	if len(remainder) > 0 {
		p.segments = append(p.segments, substring{
			value: remainder,
		})
	}

	return nil
}

func (p *pattern) capture(str string) error {
	err := p.findAllSegmentStarts(str)
	if err != nil {
		return err
	}

	starts, err := p.getTrueSegmentStarts()
	if err != nil {
		return err
	}

	for i, start := range starts {
		if i == 0 && start > 0 && p.startsWithVerb() {
			p.captures = append(p.captures, str[:start])
		}

		captureFrom := start + len(p.segments[i].value)
		var captureTo int

		if i < len(starts)-1 {
			captureTo = starts[i+1]
		} else {
			if !p.endsWithVerb() {
				break
			}

			captureTo = len(str)
		}

		capture := str[captureFrom:captureTo]
		if len(capture) > 0 {
			p.captures = append(p.captures, capture)
		}
	}

	if len(p.captures) != len(p.verbs) {
		return fmt.Errorf("%w: captured %d values for %d verbs; counts must match", ErrBug, len(p.captures), len(p.verbs))
	}

	return nil
}

func (p *pattern) findAllSegmentStarts(str string) error {
	for i := range p.segments {
		segment := p.segments[i].value
		var starts []int
		var offset int

		for offset <= len(str) {
			relativeStart := strings.Index(str[offset:], segment)
			if relativeStart < 0 {
				break
			}

			trueStart := relativeStart + offset
			starts = append(starts, trueStart)

			offset += (relativeStart + len(segment))
		}

		if len(starts) == 0 {
			return fmt.Errorf("%w: could not find substring '%s' in '%s'", ErrNoMatch, segment, str)
		}

		p.segments[i].starts = starts
	}

	return nil
}

/*
	Evaluates the list of found start indexes for each segment in the pattern
	in search of a single set, one per segment. That set locates the sequence
	of segments in the string input which perfectly match the segments in the
	pattern on either side of the verbs – the "true" segments, out of what may
	be multiple found instances of each in the string input.

	Returns ErrNoMatch if no single set is found, meaning the string input does
	not match the pattern.

	Returns ErrMultipleMatches if the string input contains more than one set
	of segments perfectly matching the pattern, making the intended captures
	ambiguous.
*/
func (p pattern) getTrueSegmentStarts() ([]int, error) {
	segCount := len(p.segments)
	var startSets [][]int

	lastSegStarts := p.segments[segCount-1].starts

	// Each start index found for the last segment in the pattern begins
	// a candidate set of segment starts. A set marks a consecutive sequence
	// of segments separated from each other only by verbs and thus perfectly
	// enclosing a series of intended captures from the string input.
	for i := range lastSegStarts {
		set := []int{lastSegStarts[i]}

		// Work backwards through each of the other segments prior to the last,
		// and backwards through each of their starts. Take the first start that,
		// combined with the segment's length, is less than the latest start added
		// to the set. This marks an instance of the segment immediately preceding
		// the one in front of it, on the other side of a verb.
		for i := segCount - 2; i >= 0; i-- {
			nextSegBack := p.segments[i]

			for j := len(nextSegBack.starts) - 1; j >= 0; j-- {
				latestStartInSet := set[len(set)-1]

				if (nextSegBack.starts[j] + len(nextSegBack.value)) < latestStartInSet {
					set = append(set, nextSegBack.starts[j])
					break
				}
			}
		}

		if len(set) == len(p.segments) {
			startSets = append(startSets, set)
		}
	}

	if len(startSets) > 1 {
		return nil, fmt.Errorf("%w: found %d; need 1", ErrMultipleMatches, len(startSets))
	}

	if len(startSets) < 1 {
		return nil, ErrNoMatch
	}

	segmentStarts := startSets[0]
	sort.Ints(segmentStarts)

	return segmentStarts, nil
}

func (p pattern) assign(targetPtrs []interface{}) error {
	for i := range targetPtrs {
		if i >= len(p.verbs) || i >= len(p.captures) {
			return fmt.Errorf("%w: no value captured to assign to 'targetPtrs[%d]'", ErrBug, i)
		}

		verb := p.verbs[i].value
		assignFunc := assignFuncs[rune(verb[1])]

		err := assignFunc(p.captures[i], targetPtrs[i])
		if err != nil {
			return fmt.Errorf("at index %d: %w", i, err)
		}
	}

	return nil
}

func (p pattern) startsWithVerb() bool {
	if len(p.verbs) > 0 {
		firstVerb := p.verbs[0]
		return firstVerb.starts[0] == 0
	}

	return false
}

func (p pattern) endsWithVerb() bool {
	if len(p.verbs) > 0 {
		lastVerb := p.verbs[len(p.verbs)-1]
		return lastVerb.starts[0] == len(p.format)-2
	}

	return false
}

func (p pattern) verbCount() int {
	return len(p.verbs)
}

// *********************************
// DEBUG METHODS
// *********************************

func (p pattern) printVerbs() {
	fmt.Print("Verbs: ")
	for i, v := range p.verbs {
		fmt.Printf("%q", v)
		if i+1 < len(p.verbs) {
			fmt.Print(", ")
		}
	}
	fmt.Println()
}

func (p pattern) printSegments() {
	fmt.Print("Segments")
	for i, s := range p.segments {
		fmt.Printf("\n  %d. %q", i+1, s)
		for i, start := range s.starts {
			if i == 0 {
				fmt.Print(" [")
			}

			fmt.Printf("%d", start)

			if i == len(s.starts)-1 {
				fmt.Printf("]")
			} else {
				fmt.Print(", ")
			}
		}
	}

	// for i, start := range p.segmentStarts {
	// 	if i == 0 {
	// 		fmt.Printf("\nFinal Segment Starts")
	// 	}

	// 	fmt.Printf("\n  %d. %q at %d", i+1, p.segments[i], start)
	// }

	fmt.Println()
}
