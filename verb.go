package main

import (
	"fmt"
	"strconv"
)

type verb struct {
	value rune
	start int
	flags []rune
}

func (v verb) String() string {
	var flags string
	for i := range v.flags {
		flags += string(v.flags[i])
	}
	return fmt.Sprintf("%%%s%c", flags, v.value)
}

func (v verb) maxWidth() (int, bool) {
	var widthFlags string
	var taking bool

	for i := range v.flags {
		f := v.flags[i]
		if f >= '0' && f <= '9' {
			taking = true
			widthFlags += string(f)
		} else {
			if taking {
				break
			}
		}
	}

	if len(widthFlags) == 0 {
		return 0, false
	}

	width, err := strconv.Atoi(widthFlags)
	if err != nil {
		return 0, false
	}

	return width, true
}
