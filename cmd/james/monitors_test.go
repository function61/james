package main

import (
	"testing"

	"github.com/function61/gokit/assert"
)

func TestEnableOrDisableMonitor(t *testing.T) {
	result, _ := enableOrDisableMonitor("1", false, getDummyMonitors())

	assert.Assert(t, len(result) == 2)
	assert.Assert(t, result[0].Enabled == false)
	assert.Assert(t, result[1].Enabled == false)

	result, _ = enableOrDisableMonitor("2", true, getDummyMonitors())

	assert.Assert(t, len(result) == 2)
	assert.Assert(t, result[0].Enabled == true)
	assert.Assert(t, result[1].Enabled == true)

	_, err := enableOrDisableMonitor("3", true, getDummyMonitors())

	assert.Assert(t, err == errUnableToFindMonitor)
}

func TestDeleteMonitor(t *testing.T) {
	result, _ := deleteMonitor("1", getDummyMonitors())

	assert.Assert(t, len(result) == 1)
	assert.EqualString(t, result[0].URL, "https://zombo.com/")

	result, _ = deleteMonitor("2", getDummyMonitors())

	assert.Assert(t, len(result) == 1)
	assert.EqualString(t, result[0].URL, "http://example.com/")

	_, err := deleteMonitor("3", getDummyMonitors())

	assert.Assert(t, err == errUnableToFindMonitor)
}

func getDummyMonitors() []Monitor {
	return []Monitor{
		{
			ID:      "1",
			Enabled: true,
			URL:     "http://example.com/",
			Find:    "foo",
		},
		{
			ID:      "2",
			Enabled: false,
			URL:     "https://zombo.com/",
			Find:    "Welcome to zombo.com",
		},
	}
}
