package main

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestEnableOrDisableMonitor(t *testing.T) {
	result, _ := enableOrDisableMonitor("1", false, getDummyMonitors())

	assert.True(t, len(result) == 2)
	assert.True(t, result[0].Enabled == false)
	assert.True(t, result[1].Enabled == false)

	result, _ = enableOrDisableMonitor("2", true, getDummyMonitors())

	assert.True(t, len(result) == 2)
	assert.True(t, result[0].Enabled == true)
	assert.True(t, result[1].Enabled == true)

	_, err := enableOrDisableMonitor("3", true, getDummyMonitors())

	assert.True(t, err == errUnableToFindMonitor)
}

func TestDeleteMonitor(t *testing.T) {
	result, _ := deleteMonitor("1", getDummyMonitors())

	assert.True(t, len(result) == 1)
	assert.EqualString(t, result[0].Url, "https://zombo.com/")

	result, _ = deleteMonitor("2", getDummyMonitors())

	assert.True(t, len(result) == 1)
	assert.EqualString(t, result[0].Url, "http://example.com/")

	_, err := deleteMonitor("3", getDummyMonitors())

	assert.True(t, err == errUnableToFindMonitor)
}

func getDummyMonitors() []Monitor {
	return []Monitor{
		{
			Id:      "1",
			Enabled: true,
			Url:     "http://example.com/",
			Find:    "foo",
		},
		{
			Id:      "2",
			Enabled: false,
			Url:     "https://zombo.com/",
			Find:    "Welcome to zombo.com",
		},
	}
}
