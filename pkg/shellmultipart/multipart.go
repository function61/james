package shellmultipart

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// TODO: check if mime/multipart can be used here

type Part struct {
	script string
	output string
}

func (p *Part) Output() string {
	return p.output
}

type Multipart struct {
	boundaryLine string
	parts        []*Part
}

func New() *Multipart {
	return &Multipart{
		boundaryLine: "-- separator e7104359-b584-4558-8a80-23aa6c4805b8",
		parts:        []*Part{},
	}
}

func (s *Multipart) AddPart(script string) *Part {
	part := &Part{
		script: script,
	}

	s.parts = append(s.parts, part)

	return part
}

func (s *Multipart) GetMultipartShellScript() string {
	aggregated := []string{}

	echoBoundaryLineCommand := fmt.Sprintf(`echo; echo "%s"`, s.boundaryLine)

	for _, part := range s.parts {
		aggregated = append(aggregated, echoBoundaryLineCommand, part.script)
	}

	aggregated = append(aggregated, echoBoundaryLineCommand)

	return strings.Join(aggregated, "\n") + "\n"
}

func (s *Multipart) ParseShellOutput(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	prevPart := ""
	i := -1
	partCount := len(s.parts)
	for scanner.Scan() {
		line := scanner.Text()
		if line == s.boundaryLine {
			if i >= 0 {
				// take out last \n
				// FIXME: this crashes if shell would not yield "\nboundary\n"
				s.parts[i].output = prevPart[0 : len(prevPart)-1]
			}
			prevPart = ""
			i++
			continue
		}

		if i >= partCount+1 {
			return errors.New("foo far, man")
		}

		prevPart += line + "\n"
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if i != partCount {
		return errors.New("did not see last boundary")
	}

	return nil
}
