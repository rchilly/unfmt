package unfmt

import (
	"fmt"
	"strings"
	"unicode"
)

const flagRunes runes = "#-. 0123456789"

func (rns runes) includes(r rune) bool {
	for _, rn := range rns {
		if rn == r {
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

func (p *pattern) reset() {
	for i := range p.segments {
		p.segments[i].starts = nil
	}

	p.trueSegmentStarts = nil
	p.captureGroups = nil
}

func unescapeFormat(format string) string {
	return strings.ReplaceAll(format, "%%", "%")
}

func (p *pattern) parseVerbs(format string) error {
	var seekVerb bool
	var flags []rune

	for idx, nextRune := range format {
		if !seekVerb {
			seekVerb = nextRune == '%'
			continue
		}

		switch {
		case nextRune == '%':
			seekVerb = false
		case flagRunes.includes(nextRune):
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
		verbIndex := strings.Index(remainder, verb.String())

		if len(remainder[:verbIndex]) > 0 {
			p.segments = append(p.segments, segment{
				value:       remainder[:verbIndex],
				formatStart: index,
			})
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

		index += verbIndex + verb.len()

		remainder = remainder[verbIndex+verb.len():]
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

		var offset int

		for offset <= len(str) {
			relativeStart := strings.Index(str[offset:], segment)
			if relativeStart < 0 {
				break
			}

			trueStart := offset + relativeStart
			p.segments[i].starts = append(p.segments[i].starts, trueStart)

			offset = trueStart + len(segment)
		}

		if len(p.segments[i].starts) == 0 {
			return fmt.Errorf("%w: could not find substring '%s' in '%s'", ErrNoMatch, segment, str)
		}
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

	lastSegmentStarts := p.segments[len(p.segments)-1].starts

	// Each start index found for the last segment in the pattern begins
	// a candidate set of segment starts. A set marks a consecutive sequence
	// of segments separated from each other only by verbs and thus perfectly
	// enclosing a series of intended captures from the string input.
	for _, lastSegmentStart := range lastSegmentStarts {
		starts := []int{lastSegmentStart}

		// Work backwards through each of the other segments prior to the last,
		// and backwards through each of their starts. Take the first start that,
		// combined with the segment's length, is less than the earliest start
		// in the set. This marks an instance of the segment immediately preceding
		// the one in front of it, on the other side of an intended capture.
		for i := len(p.segments) - 2; i >= 0; i-- {
			nextSegmentBack := p.segments[i]

			for j := len(nextSegmentBack.starts) - 1; j >= 0; j-- {
				earliestSegmentStart := starts[0]

				if (nextSegmentBack.starts[j] + len(nextSegmentBack.value)) < earliestSegmentStart {
					// Since we're working backwards from last to first segment,
					// prepend each next found start to the slice to keep it sorted.
					starts = append(starts, 0)
					copy(starts[1:], starts)
					starts[0] = nextSegmentBack.starts[j]
					break
				}
			}
		}

		if len(starts) == len(p.segments) {
			if len(p.trueSegmentStarts) > 0 {
				return ErrMultipleMatches
			}

			p.trueSegmentStarts = starts
		}
	}

	if len(p.trueSegmentStarts) < 1 {
		return ErrNoMatch
	}

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
			var from, to = -1, len(p.verbs)
			for i, v := range p.verbs {
				if v.start < segment.formatStart {
					if from < 0 {
						from = i
					}

					to = i + 1
				}
			}

			substr := str[:start]
			if len(substr) == 0 {
				return fmt.Errorf(
					"%w: expected capture at start of 'str' for leading verb(s)",
					ErrEmptyCapture,
				)
			}

			p.captureGroups = append(p.captureGroups, captureGroup{
				substr: str[:start],
				verbs:  p.verbs[from:to],
			})
		}

		// If we're on the last segment and the pattern ends with
		// a verb, assign the remaining substring after the last
		// segment's end to the remaining verb(s) before exiting
		// the loop.
		if i == startCount-1 {
			if p.endsWithVerb() {
				var from, to = -1, len(p.verbs)
				for i, v := range p.verbs {
					if v.start > segment.formatStart {
						if from < 0 {
							from = i
						}

						to = i + 1
					}
				}

				captureFrom := start + len(segments[i].value)
				substr := str[captureFrom:]
				if len(substr) == 0 {
					return fmt.Errorf(
						"%w: expected capture at end of 'str' for final verb(s)",
						ErrEmptyCapture,
					)
				}

				p.captureGroups = append(p.captureGroups, captureGroup{
					substr: str[captureFrom:],
					verbs:  p.verbs[from:to],
				})
			}

			break
		}

		// If we're not on the last segment, take the substring between where this
		// segment ends and the next one starts and assign it to a capture group with
		// any verbs between those two segments in the format.
		nextSegment := segments[i+1]
		var from, to = -1, len(p.verbs)
		for i, v := range p.verbs {
			if v.start > segment.formatStart && v.start < nextSegment.formatStart {
				if from < 0 {
					from = i
				}

				to = i + 1
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
			verbs:  p.verbs[from:to],
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

			nextSpaceIndex := strings.IndexFunc(substr, unicode.IsSpace)
			if nextSpaceIndex >= 0 && verb.stopAtSpaces() {
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
		offset := lastVerb.len()
		return lastVerb.start == len(p.format)-offset
	}

	return false
}

func (p pattern) verbCount() int {
	return len(p.verbs)
}
