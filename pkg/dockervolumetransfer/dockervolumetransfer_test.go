package dockervolumetransfer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestParseDockerVolumeLocation(t *testing.T) {
	tests := []struct {
		value string
		host  string
		name  string
	}{
		{value: "source.example.com:database", host: "source.example.com", name: "database"},
		{value: "deploy@source.example.com:database", host: "deploy@source.example.com", name: "database"},
		{value: "[2001:db8::1]:database", host: "[2001:db8::1]", name: "database"},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			location, err := parseDockerVolumeLocation(test.value)
			if err != nil {
				t.Fatalf("parseDockerVolumeLocation() error = %v", err)
			}
			if location.host != test.host || location.name != test.name {
				t.Fatalf("parseDockerVolumeLocation() = %#v, want host %q and name %q", location, test.host, test.name)
			}
		})
	}

	for _, value := range []string{"", "host", ":volume", "host:"} {
		t.Run("invalid_"+value, func(t *testing.T) {
			if _, err := parseDockerVolumeLocation(value); err == nil {
				t.Fatalf("parseDockerVolumeLocation(%q) unexpectedly succeeded", value)
			}
		})
	}
}

func TestMigrateDockerVolume(t *testing.T) {
	const archiveData = "compressed volume data"
	var calls int
	runner := func(
		ctx context.Context,
		stdin io.Reader,
		stdout io.Writer,
		stderr io.Writer,
		name string,
		args ...string,
	) error {
		calls++
		if name != "ssh" || len(args) != 3 || args[0] != "--" {
			return fmt.Errorf("unexpected command: %s %q", name, args)
		}

		switch calls {
		case 1:
			expectedCommand := "bash -c " + shellQuote(exportDockerVolumeScript) + " -- 'source-volume'"
			if args[1] != "source.example.com" || args[2] != expectedCommand {
				return fmt.Errorf("unexpected source command: %q", args)
			}
			_, err := io.WriteString(stdout, archiveData)
			return err
		case 2:
			expectedCommand := "bash -c " + shellQuote(importDockerVolumeScript) + " -- 'destination-volume'"
			if args[1] != "deploy@destination.example.com" || args[2] != expectedCommand {
				return fmt.Errorf("unexpected destination command: %q", args)
			}
			gotArchive, err := io.ReadAll(stdin)
			if err != nil {
				return err
			}
			if string(gotArchive) != archiveData {
				return fmt.Errorf("archive = %q, want %q", gotArchive, archiveData)
			}
			return nil
		default:
			return fmt.Errorf("unexpected call %d", calls)
		}
	}

	err := migrateDockerVolume(
		context.Background(),
		"source.example.com:source-volume",
		"deploy@destination.example.com:destination-volume",
		io.Discard,
		runner,
	)
	if err != nil {
		t.Fatalf("migrateDockerVolume() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("runner called %d times, want 2", calls)
	}
}

func TestMigrateDockerVolumeStopsAfterSourceFailure(t *testing.T) {
	var calls int
	runner := func(
		ctx context.Context,
		stdin io.Reader,
		stdout io.Writer,
		stderr io.Writer,
		name string,
		args ...string,
	) error {
		calls++
		return errors.New("source unavailable")
	}

	err := migrateDockerVolume(
		context.Background(),
		"source:volume",
		"destination:volume",
		io.Discard,
		runner,
	)
	if err == nil || !strings.Contains(err.Error(), "export source volume") {
		t.Fatalf("migrateDockerVolume() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("runner called %d times, want 1", calls)
	}
}

func TestEntryPointUsesPositionalLocations(t *testing.T) {
	command := EntryPoint()
	migrateCommand, _, err := command.Find([]string{"migrate"})
	if err != nil {
		t.Fatalf("find migrate command: %v", err)
	}
	if migrateCommand.Use != "migrate SOURCE DESTINATION" {
		t.Fatalf("migrate command use = %q", migrateCommand.Use)
	}
	if err := migrateCommand.Args(migrateCommand, []string{"source:volume", "destination:volume"}); err != nil {
		t.Fatalf("two positional locations rejected: %v", err)
	}
	if err := migrateCommand.Args(migrateCommand, []string{"source:volume"}); err == nil {
		t.Fatal("one positional location unexpectedly accepted")
	}
}
