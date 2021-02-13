package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	boolVal1, boolVal2, boolVal3       bool
	stringVal1, stringVal2, stringVal3 string
	intVal1, intVal2, intVal3          int
	int64Val1, int64Val2, int64Val3    int64
)

func TestGimmef(t *testing.T) {
	testCases := []struct {
		name          string
		format        string
		str           string
		targetPtrs    []interface{}
		shouldError   bool
		expectedError string
		assertResult  func(t *testing.T)
	}{
		{
			name:   "handles strings",
			format: "data-lake-ws-%s-read-%s",
			str:    "data-lake-ws-deepspace-read-yooo",
			targetPtrs: []interface{}{
				&stringVal1,
				&stringVal2,
			},
			assertResult: func(t *testing.T) {
				assert.Equal(t, "deepspace", stringVal1)
				assert.Equal(t, "yooo", stringVal2)
			},
		},
		{
			name:   "handles width and whitespace",
			format: "%5s",
			str:    "   abcdefghijk",
			targetPtrs: []interface{}{
				&stringVal1,
			},
			assertResult: func(t *testing.T) {
				assert.Equal(t, "abcde", stringVal1)
			},
		},
		// {
		// 	name:   "handles adjacent verbs",
		// 	format: "%5s%d",
		// 	str:    "   123456",
		// 	targetPtrs: []interface{}{
		// 		&stringVal1,
		// 		&intVal1,
		// 	},
		// 	assertResult: func(t *testing.T) {
		// 		assert.Equal(t, "12345", stringVal1)
		// 		assert.Equal(t, 6, intVal1)
		// 	},
		// },
		{
			name:   "handles substrings",
			format: "might contain %s fragment",
			str:    "I have a sentence that might contain      this fragment of text",
			targetPtrs: []interface{}{
				&stringVal1,
			},
			assertResult: func(t *testing.T) {
				assert.Equal(t, "this", stringVal1)
			},
		},
		{
			name:   "handles ints",
			format: "%d + %d = %d",
			str:    "1000 + -2000 = -1000",
			targetPtrs: []interface{}{
				&intVal1,
				&intVal2,
				&intVal3,
			},
			assertResult: func(t *testing.T) {
				assert.Equal(t, 1000, intVal1)
				assert.Equal(t, -2000, intVal2)
				assert.Equal(t, -1000, intVal3)
			},
		},
		{
			name:   "handles big ints",
			format: "%d + %d = %d",
			str:    "10000000000 + 20000000000 = 30000000000",
			targetPtrs: []interface{}{
				&int64Val1,
				&int64Val2,
				&int64Val3,
			},
			assertResult: func(t *testing.T) {
				assert.Equal(t, int64(1e10), int64Val1)
				assert.Equal(t, int64(2e10), int64Val2)
				assert.Equal(t, int64(3e10), int64Val3)
			},
		},
		{
			name:   "handles bools",
			format: "employed: %t, retired: %t, part-time: %t",
			str:    "employed: 1, retired: FALSE, part-time: f",
			targetPtrs: []interface{}{
				&boolVal1,
				&boolVal2,
				&boolVal3,
			},
			assertResult: func(t *testing.T) {
				assert.Equal(t, true, boolVal1)
				assert.Equal(t, false, boolVal2)
				assert.Equal(t, false, boolVal3)
			},
		},
		{
			name:   "handles %%",
			format: "%d%% of %d is %d",
			str:    "50% of 100 is 50",
			targetPtrs: []interface{}{
				&intVal1,
				&intVal2,
				&intVal3,
			},
			assertResult: func(t *testing.T) {
				assert.Equal(t, 50, intVal1)
				assert.Equal(t, 100, intVal2)
				assert.Equal(t, 50, intVal3)
			},
		},
		{
			name:   "handles multiple candidates",
			format: "and a %d and a %d and a %d!",
			str:    "and a 1 and a 2 and 3! and a 2 and a 3 and a 4!",
			targetPtrs: []interface{}{
				&intVal1,
				&intVal2,
				&intVal3,
			},
			assertResult: func(t *testing.T) {
				assert.Equal(t, 2, intVal1)
				assert.Equal(t, 3, intVal2)
				assert.Equal(t, 4, intVal3)
			},
		},
		{
			name:   "returns error for unsupported verb",
			format: "%s was a very good %z",
			str:    "fido was a very good boy",
			targetPtrs: []interface{}{
				&intVal1,
				&intVal2,
			},
			shouldError:   true,
			expectedError: fmt.Sprintf("parsing 'format': %s: unsupported verb '%%z'", ErrBadArg),
		},
		{
			name:   "returns error for consecutive instances of same verb",
			format: "two numbers %d%d went for a walk",
			str:    "two numbers 100200 went for a walk",
			targetPtrs: []interface{}{
				&intVal1,
				&intVal2,
			},
			shouldError:   true,
			expectedError: fmt.Sprintf("parsing 'format': %s: found consecutive instances of verb '%%d' without an intervening substring", ErrBadArg),
		},
		{
			name:   "returns ErrNoMatch for not all substrings found",
			format: `"What a beautiful %s!" said %s.`,
			str:    `"What a beautiful hot air balloon?" said Heidi.`,
			targetPtrs: []interface{}{
				&stringVal1,
				&stringVal2,
			},
			shouldError:   true,
			expectedError: fmt.Sprintf(`applying 'format' to 'str': %s: could not find substring '!" said ' in '"What a beautiful hot air balloon?" said Heidi.'`, ErrNoMatch),
		},
		{
			name:   "returns ErrNoMatch for all substrings found but not in order",
			format: "I want %s that way",
			str:    "just that way, I want it",
			targetPtrs: []interface{}{
				&stringVal1,
			},
			shouldError:   true,
			expectedError: fmt.Sprintf("applying 'format' to 'str': %s", ErrNoMatch),
		},
		{
			name:   "returns ErrNoMatch for all substrings found in order but too few captures",
			format: "and a %d and a %d and a %d!",
			str:    "and a 1 and a 2 and a !",
			targetPtrs: []interface{}{
				&intVal1,
				&intVal2,
				&intVal3,
			},
			shouldError:   true,
			expectedError: fmt.Sprintf("applying 'format' to 'str': %s", ErrNoMatch),
		},
		{
			name:   "returns ErrMultipleMatches for multiple matches",
			format: "and a %d and a %d and a %d!",
			str:    "and a 1 and a 2 and a 3! and a 2 and a 3 and a 4!",
			targetPtrs: []interface{}{
				&intVal1,
				&intVal2,
				&intVal3,
			},
			shouldError:   true,
			expectedError: fmt.Sprintf("applying 'format' to 'str': %s: found 2; need 1", ErrMultipleMatches),
		},
		{
			name:   "returns error for wrong target type",
			format: "Gimme a %s, any %s",
			str:    "Gimme a number, any number",
			targetPtrs: []interface{}{
				&stringVal1,
				&intVal1,
			},
			shouldError:   true,
			expectedError: "assigning values to 'targetPtrs': at index 1: expected string pointer as target, got *int",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Gimmef(tc.format, tc.str, tc.targetPtrs...)
			if tc.shouldError {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.expectedError)
				return
			}

			assert.NoError(t, err)
			tc.assertResult(t)
		})
	}
}

const story = `Once upon a time, there was a cat named Lola. 
She liked to curl up in our yard. 
Her favorite color is yellow and her favorite number is 3, but that's silly, because she's a cat.`

func BenchmarkGimmef(b *testing.B) {
	var favoriteNumber string
	var three int

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := Gimmef("and her %s is %d,", story, &favoriteNumber, &three)
		if err != nil {
			b.Fatal("got unexpected error", err)
		}
	}

	b.StopTimer()

	assert.Equal(b, "favorite number", favoriteNumber)
	assert.Equal(b, 3, three)
}

func TestFormatting(t *testing.T) {
	t.SkipNow()

	if false {
		t.FailNow()
	}
	fmt.Printf("%0120.2f\n", 10.123)
}

func TestVerbs(t *testing.T) {
	t.SkipNow()

	if false {
		t.FailNow()
	}

	format := "%5s %9.2d %-8s"

	p, err := newPattern(format)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("p.segments = %#v\n", p.segments)
	fmt.Printf("p.verbs = %#v\n", p.verbs)
}
