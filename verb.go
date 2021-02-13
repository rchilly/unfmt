package main

import (
	"fmt"
	"strconv"
	"strings"
)

type verb struct {
	value rune
	start int
	flags []rune
}

func (v verb) String() string {
	var flags []string
	for i := range v.flags {
		flags = append(flags, string(v.flags[i]))
	}
	return fmt.Sprintf("%c%s%c", pct, strings.Join(flags, ""), v.value)
}

func (v verb) maxWidth() (int, bool) {
	if v.value != verbString {
		return 0, false
	}

	var widthFlags []string
	var taking bool

	for i := range v.flags {
		f := v.flags[i]
		if f >= 48 && f <= 57 {
			taking = true
			widthFlags = append(widthFlags, string(f))
		} else {
			if taking {
				break
			}
		}
	}

	if len(widthFlags) == 0 {
		return 0, false
	}

	width, err := strconv.Atoi(strings.Join(widthFlags, ""))
	if err != nil {
		return 0, false
	}

	return width, true
}
