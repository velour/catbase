package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetGet(t *testing.T) {
	cfg := ReadConfig(":memory:", ":memory:")
	expected := "value"
	cfg.Set("test", expected)
	actual := cfg.Get("test", "NOPE")
	assert.Equal(t, expected, actual, "Config did not store values")
}

func TestSetGetArray(t *testing.T) {
	cfg := ReadConfig(":memory:", ":memory:")
	expected := []string{"a", "b", "c"}
	cfg.SetArray("test", expected)
	actual := cfg.GetArray("test", []string{"NOPE"})
	assert.Equal(t, expected, actual, "Config did not store values")
}
