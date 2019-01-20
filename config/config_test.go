package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetGet(t *testing.T) {
	cfg := ReadConfig(":memory:")
	expected := "value"
	cfg.Set("test", expected)
	actual := cfg.Get("test")
	assert.Equal(t, expected, actual, "Config did not store values")
}

func TestSetGetArray(t *testing.T) {
	cfg := ReadConfig(":memory:")
	expected := []string{"a", "b", "c"}
	cfg.SetArray("test", expected)
	actual := cfg.GetArray("test")
	assert.Equal(t, expected, actual, "Config did not store values")
}
