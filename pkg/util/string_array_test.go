package util

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestStringArrayString(t *testing.T) {
	sa := StringArray{}
	assert.Equal(t, "", sa.String())
	err := sa.Set("foo")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	assert.Equal(t, "foo", sa.String())
	err = sa.Set("bar")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	assert.Equal(t, "foo,bar", sa.String())
}

func TestStringArrayGet(t *testing.T) {
	sa := StringArray{}
	value := sa.Get()
	assert.Equal(t, []string{}, value)

	err := sa.Set("foo")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	value = sa.Get()
	assert.Equal(t, []string{"foo"}, value)
}
