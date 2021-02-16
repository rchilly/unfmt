package main

import (
	"fmt"
	"strconv"
	"strings"
)

type assignFunc func(string, interface{}) (int, error)

var (
	boolRunes = "01truefalseTRUEFALSE"
	intRunes  = "+-0123456789"

	assignFuncs = map[rune]assignFunc{
		verbBool:   assignBool,
		verbString: assignString,
		verbInt:    assignInt,
	}
)

func isSupportedVerb(r rune) bool {
	_, ok := assignFuncs[r]
	return ok
}

// TODO: Make this setup generic i.e. DRY.
func isNonBooly(r rune) bool {
	for _, br := range boolRunes {
		if br == r {
			return false
		}
	}

	return true
}

func assignBool(str string, target interface{}) (int, error) {
	pBool, ok := target.(*bool)
	if !ok {
		return 0, fmt.Errorf("expected bool pointer as target, got %T", target)
	}

	switch nonBoolIndex := strings.IndexFunc(str, isNonBooly); nonBoolIndex {
	case 0:
		return 0, fmt.Errorf("expected one or more leading boolean characters, got '%s'", str)
	case -1:
	default:
		str = str[:nonBoolIndex]
	}

	b, err := strconv.ParseBool(str)
	if err != nil {
		return 0, fmt.Errorf("error converting '%s' to bool: %w", str, err)
	}

	*pBool = b
	return len(str), nil
}

func assignString(str string, target interface{}) (int, error) {
	pStr, ok := target.(*string)
	if !ok {
		return 0, fmt.Errorf("expected string pointer as target, got %T", target)
	}

	*pStr = str
	return len(str), nil
}

func isNonInty(r rune) bool {
	for _, ir := range intRunes {
		if ir == r {
			return false
		}
	}

	return true
}

func assignInt(str string, target interface{}) (int, error) {
	var signed int64
	var unsigned uint64
	var err error

	switch nonIntIndex := strings.IndexFunc(str, isNonInty); nonIntIndex {
	case 0:
		return 0, fmt.Errorf("expected one or more leading numeric characters, got '%s'", str)
	case -1:
	default:
		str = str[:nonIntIndex]
	}

	switch v := target.(type) {
	case *int:
		signed, err = strconv.ParseInt(str, 10, 0)
		*v = int(signed)
	case *int8:
		signed, err = strconv.ParseInt(str, 10, 8)
		*v = int8(signed)
	case *int16:
		signed, err = strconv.ParseInt(str, 10, 16)
		*v = int16(signed)
	case *int32:
		signed, err = strconv.ParseInt(str, 10, 32)
		*v = int32(signed)
	case *int64:
		signed, err = strconv.ParseInt(str, 10, 64)
		*v = signed
	case *uint:
		unsigned, err = strconv.ParseUint(str, 10, 0)
		*v = uint(unsigned)
	case *uint8:
		unsigned, err = strconv.ParseUint(str, 10, 8)
		*v = uint8(unsigned)
	case *uint16:
		unsigned, err = strconv.ParseUint(str, 10, 16)
		*v = uint16(unsigned)
	case *uint32:
		unsigned, err = strconv.ParseUint(str, 10, 32)
		*v = uint32(unsigned)
	case *uint64:
		unsigned, err = strconv.ParseUint(str, 10, 64)
		*v = unsigned
	default:
		return 0, fmt.Errorf("expected integer pointer as target, got %T", target)
	}

	if err != nil {
		return 0, fmt.Errorf("error converting '%s' to integer: %w", str, err)
	}

	return len(str), nil
}
