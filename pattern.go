package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var flags = []rune{
	'#',
	'-',
	'.',
	' ',
	'0',
	'1',
	'2',
	'3',
	'4',
	'5',
	'6',
	'7',
	'8',
	'9',
}

func isFlag(r rune) bool {
	for _, f := range flags {
		if f == r {
			return true
		}
	}

	return false
}

type pattern struct {
	format            string
	verbs             []verb
	segments          []segment
	trueSegmentStarts []int
	captureGroups     []captureGroup
}

type captureGroup struct {
	substr string
	verbs  []verb
}

type segment struct {
	value       string
	formatStart int
	starts      []int
}

func newPattern(format string) (p pattern, err error) {
	err = p.parseVerbs(format)
	if err != nil {
		return
	}

	p.format = format

	// After parsing verbs, must unescape '%%'s before parsing segments
	// in order to match literal '%'s in the string input.
	err = p.parseSegments(unescapeFormat(format))
	return
}

func unescapeFormat(format string) string {
	return strings.ReplaceAll(format, "%%", "%")
}

func (p *pattern) parseVerbs(format string) error {
	fmtBytes := []byte(format)

	var seekVerb bool
	var idx int
	var flags []rune

	for len(fmtBytes) > 0 {
		nextRune, size := utf8.DecodeRune(fmtBytes)

		if seekVerb {
			switch {
			case nextRune == '%':
				seekVerb = false
			case isFlag(nextRune):
				flags = append(flags, nextRune)
			case isSupportedVerb(nextRune):
				offset := len("%") + len(flags)
				p.verbs = append(p.verbs, verb{
					start: idx - offset,
					value: nextRune,
					flags: flags,
				})

				seekVerb = false

				flags = nil
			default:
				return fmt.Errorf("%w: unsupported verb '%s'", ErrBadArg, verb{
					value: nextRune,
					flags: flags,
				})
			}
		} else if nextRune == '%' {
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

The number of segments will vary, depending on the position
of verbs in the format string. For example:

Format 'my-%s-format-%s-string-%s-rocks' yields 4 segments
'my-', '-format-', '-string-', and '-rocks' for 3 verbs.

Format '%d + %d = %d' yields 2 segments ' + ' and ' = ',
also for 3 verbs.

Format '%5s%d' yields 0 segments for 2 verbs.

Expects any literal '%'s in the format to be unescaped.
*/
func (p *pattern) parseSegments(unescapedFormat string) error {
	maxSegments := len(p.verbs) + 1
	p.segments = make([]segment, 0, maxSegments)

	remainder := unescapedFormat
	index := 0

	for i, verb := range p.verbs {
		halves := strings.SplitN(remainder, verb.String(), 2)
		if len(halves[0]) > 0 {
			p.segments = append(p.segments, segment{
				value:       halves[0],
				formatStart: index,
			})

			index += len(halves[0])
		} else if i > 0 {
			previousVerb := p.verbs[i-1]
			if verb.value == previousVerb.value {
				if _, ok := previousVerb.maxWidth(); !ok {
					return fmt.Errorf(
						"%w: found consecutive instances of verb '%%%c' without a max width or intervening substring",
						ErrBadArg,
						verb.value,
					)
				}
			}
		}

		remainder = halves[1]

		index += len(verb.String())
	}

	if len(remainder) > 0 {
		p.segments = append(p.segments, segment{
			value:       remainder,
			formatStart: index,
		})
	}

	return nil
}

// TODO: Update me to take any other capture-limiting flags into account besides max width.
func (p *pattern) capture(str string) error {
	err := p.findAllSegmentStarts(str)
	if err != nil {
		return err
	}

	err = p.getTrueSegmentStarts()
	if err != nil {
		return err
	}

	err = p.getCaptureGroups(str)
	if err != nil {
		return err
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

			trueStart := offset + relativeStart
			starts = append(starts, trueStart)

			offset = trueStart + len(segment)
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
in search of a single set, one index per segment. That set locates the sequence
of segments in the string input which perfectly matches the segments in the
pattern on either side of the verbs – the "true" segments, out of what may
be multiple found instances of each in the string input.

Returns ErrNoMatch if no single set is found, meaning the string input does
not match the pattern.

Returns ErrMultipleMatches if the string input contains more than one set
of segments perfectly matching the pattern, making the intended captures
ambiguous.
*/
func (p *pattern) getTrueSegmentStarts() error {
	if len(p.segments) == 0 {
		return nil
	}

	var startSets [][]int

	lastSegmentStarts := p.segments[len(p.segments)-1].starts

	// Each start index found for the last segment in the pattern begins
	// a candidate set of segment starts. A set marks a consecutive sequence
	// of segments separated from each other only by verbs and thus perfectly
	// enclosing a series of intended captures from the string input.
	for _, lastSegmentStart := range lastSegmentStarts {
		set := []int{lastSegmentStart}

		// Work backwards through each of the other segments prior to the last,
		// and backwards through each of their starts. Take the first start that,
		// combined with the segment's length, is less than the latest start added
		// to the set. This marks an instance of the segment immediately preceding
		// the one in front of it, on the other side of an intended capture.
		for i := len(p.segments) - 2; i >= 0; i-- {
			nextSegmentBack := p.segments[i]

			for j := len(nextSegmentBack.starts) - 1; j >= 0; j-- {
				latestStartInSet := set[len(set)-1]

				if (nextSegmentBack.starts[j] + len(nextSegmentBack.value)) < latestStartInSet {
					set = append(set, nextSegmentBack.starts[j])
					break
				}
			}
		}

		if len(set) == len(p.segments) {
			startSets = append(startSets, set)
		}
	}

	if len(startSets) > 1 {
		return fmt.Errorf("%w: found %d; need 1", ErrMultipleMatches, len(startSets))
	}

	if len(startSets) < 1 {
		return ErrNoMatch
	}

	p.trueSegmentStarts = startSets[0]
	sort.Ints(p.trueSegmentStarts)

	return nil
}

// TODO: Split into more readable parts? Or at least comment.
func (p *pattern) getCaptureGroups(str string) error {
	segments := p.segments
	starts := p.trueSegmentStarts

	// If no segment starts, then the format consists only of
	// one or more verbs, against which the whole string should
	// be evaluated.
	if len(starts) == 0 {
		p.captureGroups = append(p.captureGroups, captureGroup{
			substr: str,
			verbs:  p.verbs,
		})
	}

	startCount := len(starts)
	for i, start := range starts {
		segment := segments[i]

		// If we're on the first segment start and the pattern
		// begins with a verb, assign the substring up until
		// that start to the verb(s) before that segment's start
		// in the format.
		if i == 0 && p.beginsWithVerb() {
			var verbs []verb
			for _, v := range p.verbs {
				if v.start < segment.formatStart {
					verbs = append(verbs, v)
				}
			}

			substr := str[:start]
			if len(substr) == 0 {
				return fmt.Errorf(
					"%w: expected capture at start of 'str' for leading verb '%s'",
					ErrEmptyCapture,
					p.verbs[0],
				)
			}

			p.captureGroups = append(p.captureGroups, captureGroup{
				substr: str[:start],
				verbs:  verbs,
			})
		}

		// If we're on the last segment and the pattern ends with
		// a verb, assign the remaining substring after the last
		// segment's end to the remaining verb(s) before exiting
		// the loop.
		if i == startCount-1 {
			if p.endsWithVerb() {
				var verbs []verb
				for _, v := range p.verbs {
					if v.start > segment.formatStart {
						verbs = append(verbs, v)
					}
				}

				captureFrom := start + len(segments[i].value)
				substr := str[captureFrom:]
				if len(substr) == 0 {
					return fmt.Errorf(
						"%w: expected capture at end of 'str' for final verb '%s'",
						ErrEmptyCapture,
						p.verbs[len(p.verbs)-1],
					)
				}

				p.captureGroups = append(p.captureGroups, captureGroup{
					substr: str[captureFrom:],
					verbs:  verbs,
				})
			}

			break
		}

		// If we're not on the last segment, take the substring between where this
		// segment ends and the next one starts and assign it to a capture group with
		// any verbs between those two segments in the format.
		nextSegment := segments[i+1]
		var verbs []verb
		for _, v := range p.verbs {
			if v.start > segment.formatStart && v.start < nextSegment.formatStart {
				verbs = append(verbs, v)
			}
		}

		captureFrom := start + len(segments[i].value)
		captureTo := starts[i+1]
		substr := str[captureFrom:captureTo]
		if len(substr) == 0 {
			return fmt.Errorf(
				"%w: no string to capture between matching segments '%s' and '%s', so pattern should not have matched",
				ErrBug,
				segment.value,
				nextSegment.value,
			)
		}

		p.captureGroups = append(p.captureGroups, captureGroup{
			substr: str[captureFrom:captureTo],
			verbs:  verbs,
		})
	}

	for _, group := range p.captureGroups {
		if len(group.verbs) == 0 {
			return fmt.Errorf("%w: no verbs assigned to captured substring '%s'", ErrBug, group.substr)
		}
	}

	return nil
}

func (p pattern) assign(targetPtrs []interface{}) error {
	targetPtrsIndex := 0
	for _, group := range p.captureGroups {

		var err error
		substr := group.substr

		for _, verb := range group.verbs {
			if len(targetPtrs) <= targetPtrsIndex {
				err = fmt.Errorf(
					"%w: no element found at 'targetPtrs[%d]' for next verb '%s' and substring '%s'",
					ErrBug,
					targetPtrsIndex,
					verb,
					substr,
				)

				break
			}

			if len(substr) == 0 {
				err = fmt.Errorf(
					"all of substring '%s' consumed by prior adjacent verb(s), none left for next verb '%s'",
					group.substr,
					verb,
				)

				break
			}

			substr = strings.TrimLeftFunc(substr, unicode.IsSpace)

			// For this next value to be assigned, evaluate the full remaining substring with two
			// exceptions. If it contains a space character, stop evaluation there. And if this verb
			// specifies a max width less than the length of the remaining substring or less than the
			// index of the next space character, only take that much of the substring.
			stopEvaluateIndex := len(substr)
			if nextSpaceIndex := strings.IndexFunc(substr, unicode.IsSpace); nextSpaceIndex >= 0 {
				stopEvaluateIndex = nextSpaceIndex
			}
			if maxWidth, ok := verb.maxWidth(); ok && maxWidth < stopEvaluateIndex {
				stopEvaluateIndex = maxWidth
			}

			assignFunc := assignFuncs[verb.value]

			var n int
			n, err = assignFunc(substr[:stopEvaluateIndex], targetPtrs[targetPtrsIndex])
			if err != nil {
				break
			}

			// Before the next verb, re-slice the substring to start wherever evaluation
			// stopped for this latest assignment, which may be less than the index above
			// if fewer bytes of that string were actually evaluated than those passed.
			if n < stopEvaluateIndex {
				stopEvaluateIndex = n
			}

			substr = substr[stopEvaluateIndex:]

			targetPtrsIndex++
		}

		if err != nil {
			return fmt.Errorf("at index %d: %w", targetPtrsIndex, err)
		}
	}

	return nil
}

func (p pattern) beginsWithVerb() bool {
	if len(p.verbs) > 0 {
		firstVerb := p.verbs[0]
		return firstVerb.start == 0
	}

	return false
}

func (p pattern) endsWithVerb() bool {
	if len(p.verbs) > 0 {
		lastVerb := p.verbs[len(p.verbs)-1]
		offset := len(lastVerb.String())
		return lastVerb.start == len(p.format)-offset
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

	fmt.Println()
}

func (p pattern) printCaptureGroups(str string) {
	fmt.Println("FOR")
	fmt.Printf("\tFormat = %s\n", p.format)
	fmt.Printf("\tString = %s\n", str)
	fmt.Println("WITH SEGMENTS")

	for _, s := range p.segments {
		fmt.Printf("\tSegment: %s, Format Start: %d\n", s.value, s.formatStart)
	}

	fmt.Println("CAPTURE MAP")
	for _, group := range p.captureGroups {
		fmt.Printf("\tCapture: %s, Verbs: %v\n", group.substr, group.verbs)
	}

	fmt.Println()
}
