package controllers

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortByReversedS3objects1(t *testing.T) {
	expected := []string{"3", "2", "1"}

	actual := []string{"3", "1", "2"}
	sort.Sort(reversedS3objects{(actual))

	assert.Equal(t, expected, actual)
}

func TestSortByReversedS3objects2(t *testing.T) {
	expected := []string{"/20", "/10", "/101"}

	actual := []string{"/20", "/101", "/10"}
	sort.Sort(reversedS3objects(actual))

	assert.Equal(t, expected, actual)
}

func TestSortByReversedS3objects3(t *testing.T) {
	expected := []string{"/200/10", "/101/1", "/10/2"}

	actual := []string{"/200/10", "/10/2", "/101/1"}
	sort.Sort(reversedS3objects(actual))

	assert.Equal(t, expected, actual)
}
