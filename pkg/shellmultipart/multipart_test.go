package shellmultipart

import (
	"bytes"
	"github.com/function61/gokit/assert"
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	s := New()

	hostnameRun := s.AddPart("hostname")
	osReleaseRun := s.AddPart("cat /etc/os-release")
	noNewlineRun := s.AddPart("echo -n noNewline")
	zeroContentRun := s.AddPart("")

	expectedScript := `echo; echo "-- separator e7104359-b584-4558-8a80-23aa6c4805b8"
hostname
echo; echo "-- separator e7104359-b584-4558-8a80-23aa6c4805b8"
cat /etc/os-release
echo; echo "-- separator e7104359-b584-4558-8a80-23aa6c4805b8"
echo -n noNewline
echo; echo "-- separator e7104359-b584-4558-8a80-23aa6c4805b8"

echo; echo "-- separator e7104359-b584-4558-8a80-23aa6c4805b8"
`

	assert.EqualString(t, s.GetMultipartShellScript(), expectedScript)

	// pretend to run SSH, which yields to buffer
	pretendAnswer := `
some heading noise here..
-- separator e7104359-b584-4558-8a80-23aa6c4805b8
my-hostname.example.com

-- separator e7104359-b584-4558-8a80-23aa6c4805b8
NAME="Container Linux by CoreOS"
ID=coreos
VERSION=1911.5.0
VERSION_ID=1911.5.0
BUILD_ID=2018-12-15-2317
PRETTY_NAME="Container Linux by CoreOS 1911.5.0 (Rhyolite)"
ANSI_COLOR="38;5;75"
HOME_URL="https://coreos.com/"
BUG_REPORT_URL="https://issues.coreos.com"
COREOS_BOARD="amd64-usr"

-- separator e7104359-b584-4558-8a80-23aa6c4805b8
noNewline
-- separator e7104359-b584-4558-8a80-23aa6c4805b8

-- separator e7104359-b584-4558-8a80-23aa6c4805b8
some trailing noise here..
`
	assert.Assert(t, s.ParseShellOutput(bytes.NewBufferString(pretendAnswer)) == nil)

	assert.EqualString(t, hostnameRun.Output(), "my-hostname.example.com\n")
	assert.Assert(t, strings.HasPrefix(osReleaseRun.Output(), `NAME="Container Linux by CoreOS"`))

	assert.EqualString(t, noNewlineRun.Output(), "noNewline")

	assert.EqualString(t, zeroContentRun.Output(), "")
}

func TestError(t *testing.T) {
	s := New()

	_ = s.AddPart("hostname")

	pretendAnswer := `
some heading noise here..
-- separator e7104359-b584-4558-8a80-23aa6c4805b8
this output will not have closing separator
`

	err := s.ParseShellOutput(bytes.NewBufferString(pretendAnswer))
	assert.Assert(t, err != nil)
	assert.EqualString(t, err.Error(), "did not see last boundary")
}
