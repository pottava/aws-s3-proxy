package controllers

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortByS3objects1(t *testing.T) {
	expected := []string{"1", "2", "3"}

	actual := []string{"3", "1", "2"}
	sort.Sort(s3objects(actual))

	assert.Equal(t, expected, actual)
}

func TestSortByS3objects2(t *testing.T) {
	expected := []string{"/101", "/10", "/20"}

	actual := []string{"/20", "/101", "/10"}
	sort.Sort(s3objects(actual))

	assert.Equal(t, expected, actual)
}

func TestSortByS3objects3(t *testing.T) {
	expected := []string{"/10/2", "/101/1", "/200/10"}

	actual := []string{"/200/10", "/10/2", "/101/1"}
	sort.Sort(s3objects(actual))

	assert.Equal(t, expected, actual)
}
