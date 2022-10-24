package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewSet(t *testing.T) {
	s := NewSet[string]()
	assert.Equal(t, 0, s.Size())
}

func TestSetHas(t *testing.T) {
	s := NewSet[string]()
	s.Add("foo")
	assert.True(t, s.Has("foo"))
	assert.False(t, s.Has("bar"))
}

func TestSetAdd(t *testing.T) {
	s := NewSet[string]()
	s.Add("foo")
	assert.True(t, s.Has("foo"))
}

func TestSetRemove(t *testing.T) {
	s := NewSet[string]()
	s.Add("foo")
	s.Remove("foo")
	assert.False(t, s.Has("foo"))
}

func TestSetClear(t *testing.T) {
	s := NewSet[string]()
	s.Add("foo")
	s.Clear()
	assert.False(t, s.Has("foo"))
}

func TestSetSize(t *testing.T) {
	s := NewSet[string]()
	s.Add("foo")
	assert.Equal(t, 1, s.Size())
}

func TestSetEmptyWithoutInit(t *testing.T) {
	var s Set[string]
	assert.Equal(t, 0, s.Size())
}

func TestSetSlice(t *testing.T) {
	s := NewSet[string]()
	s.Add("foo")
	s.Add("bar")
	assert.ElementsMatch(t, []string{"foo", "bar"}, s.Slice())

	s.Remove("foo")
	assert.ElementsMatch(t, []string{"bar"}, s.Slice())
}
