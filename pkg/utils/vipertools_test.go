package utils_test

import (
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/utils"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstNonEmptyBool(t *testing.T) {
	v := viper.New()
	v.Set("second", false)
	v.Set("third", true)

	value := utils.FirstNonEmptyBool(v, "first", "second", "third")
	assert.False(t, value)
}

func TestFirstNonEmptyBool_NonBool(t *testing.T) {
	v := viper.New()
	v.Set("first", "stringvalue")

	value := utils.FirstNonEmptyBool(v, "first")
	assert.False(t, value)
}

func TestFirstNonEmptyBool_NilPointer(t *testing.T) {
	value := utils.FirstNonEmptyBool(nil, "first")
	assert.False(t, value)
}

func TestFirstNonEmptyBool_EmptyKeys(t *testing.T) {
	v := viper.New()
	value := utils.FirstNonEmptyBool(v)
	assert.False(t, value)
}

func TestFirstNonEmptyBool_NotFound(t *testing.T) {
	value := utils.FirstNonEmptyBool(viper.New(), "key")
	assert.False(t, value)
}

func TestFirstNonEmptyInt(t *testing.T) {
	v := viper.New()
	v.Set("second", 42)
	v.Set("third", 99)

	value, ok := utils.FirstNonEmptyInt(v, "first", "second", "third")
	require.True(t, ok)

	assert.Equal(t, 42, value)
}

func TestFirstNonEmptyInt_NilPointer(t *testing.T) {
	_, ok := utils.FirstNonEmptyInt(nil, "first")
	assert.False(t, ok)
}

func TestFirstNonEmptyInt_EmptyKeys(t *testing.T) {
	v := viper.New()
	value, ok := utils.FirstNonEmptyInt(v)
	require.False(t, ok)

	assert.Zero(t, value)
}

func TestFirstNonEmptyInt_NotFound(t *testing.T) {
	value, ok := utils.FirstNonEmptyInt(viper.New(), "key")
	require.False(t, ok)

	assert.Zero(t, value)
}

func TestFirstNonEmptyInt_EmptyInt(t *testing.T) {
	v := viper.New()
	v.Set("first", 0)

	value, ok := utils.FirstNonEmptyInt(v, "first")
	assert.True(t, ok)

	assert.Zero(t, value)
}

func TestFirstNonEmptyInt_StringValue(t *testing.T) {
	v := viper.New()
	v.Set("first", "stringvalue")

	value, ok := utils.FirstNonEmptyInt(v, "first")
	require.False(t, ok)

	assert.Zero(t, value)
}

func TestFirstNonEmptyString(t *testing.T) {
	v := viper.New()
	v.Set("second", "secret")
	v.Set("third", "ignored")

	value := utils.FirstNonEmptyString(v, "first", "second", "third")
	assert.Equal(t, "secret", value)
}

func TestFirstNonEmptyString_Empty(t *testing.T) {
	v := viper.New()
	v.Set("second", "")
	v.Set("third", "secret")

	value := utils.FirstNonEmptyString(v, "first", "second", "third")
	assert.Empty(t, value)
}

func TestFirstNonEmptyString_NilPointer(t *testing.T) {
	value := utils.FirstNonEmptyString(nil, "first")
	assert.Empty(t, value)
}

func TestFirstNonEmptyString_EmptyKeys(t *testing.T) {
	v := viper.New()
	value := utils.FirstNonEmptyString(v)
	assert.Empty(t, value)
}

func TestFirstNonEmptyString_NotFound(t *testing.T) {
	value := utils.FirstNonEmptyString(viper.New(), "key")
	assert.Empty(t, value)
}

func TestGetString(t *testing.T) {
	v := viper.New()
	v.Set("some", "value")

	value := utils.GetString(v, "some")
	assert.Equal(t, "value", value)
}

func TestGetString_DoubleQuotes(t *testing.T) {
	v := viper.New()
	v.Set("some", "\"value\"")

	value := utils.GetString(v, "some")
	assert.Equal(t, "value", value)
}

func TestGetStringMapString(t *testing.T) {
	v := viper.New()
	v.Set("settings.github.com/wakatime", "value")
	v.Set("settings_foo.debug", "true")

	expected := map[string]string{
		"github.com/wakatime": "value",
	}

	m := utils.GetStringMapString(v, "settings")
	assert.Equal(t, expected, m)
}

func TestGetStringMapString_NotFound(t *testing.T) {
	v := viper.New()
	v.Set("settings.key", "value")

	m := utils.GetStringMapString(v, "internal")
	assert.Equal(t, map[string]string{}, m)
}
