package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanner_ScanString(t *testing.T) {
	scanner, err := NewScanner("%5s%d")
	if err != nil {
		t.Fatal(err)
	}

	var str string
	var i int

	err = scanner.ScanString("f00 22", &str, &i)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "f00", str)
	assert.Equal(t, 22, i)

	err = scanner.ScanString("foo221000", &str, &i)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "foo22", str)
	assert.Equal(t, 1000, i)

	err = scanner.ScanString("blue 42 set hut hut!", &str, &i)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "blue", str)
	assert.Equal(t, 42, i)
}
