package main

import (
	"fmt"
	"strconv"
)

type assignFunc func(string, interface{}) error

var assignFuncs = map[rune]assignFunc{
	verbBool:   assignBool,
	verbString: assignString,
	verbInt:    assignInt,
}

func isSupportedVerb(r rune) bool {
	_, ok := assignFuncs[r]
	return ok
}

func assignBool(str string, target interface{}) error {
	pBool, ok := target.(*bool)
	if !ok {
		return fmt.Errorf("expected bool pointer as target, got %T", target)
	}

	b, err := strconv.ParseBool(str)
	if err != nil {
		return fmt.Errorf("error converting '%s' to bool: %w", str, err)
	}

	*pBool = b
	return nil
}

func assignString(str string, target interface{}) error {
	pStr, ok := target.(*string)
	if !ok {
		return fmt.Errorf("expected string pointer as target, got %T", target)
	}

	*pStr = str
	return nil
}

func assignInt(str string, target interface{}) error {
	var signed int64
	var unsigned uint64
	var err error

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
		return fmt.Errorf("expected integer pointer as target, got %T", target)
	}

	if err != nil {
		return fmt.Errorf("error converting '%s' to integer: %w", str, err)
	}

	return nil
}
