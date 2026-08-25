// Tool for migrating Docker volume source host to new host
package dockervolumetransfer

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func EntryPoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Docker volume related commands",
	}

	migrateCmd := &cobra.Command{
		Use:   "migrate SOURCE DESTINATION",
		Short: "Migrate a Docker volume; SOURCE and DESTINATION use [user@]host:volumename format",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrateDockerVolume(
				cmd.Context(),
				args[0],
				args[1],
				cmd.ErrOrStderr(),
				nativeVolumeCommandRunner,
			)
		},
	}

	cmd.AddCommand(migrateCmd)
	return cmd
}

func migrateDockerVolume(
	ctx context.Context,
	sourceValue string,
	destinationValue string,
	stderr io.Writer,
	runner volumeCommandRunner,
) error {
	source, err := parseDockerVolumeLocation(sourceValue)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	destination, err := parseDockerVolumeLocation(destinationValue)
	if err != nil {
		return fmt.Errorf("destination: %w", err)
	}

	archive, err := os.CreateTemp("", "james-volume-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temporary archive: %w", err)
	}
	archivePath := archive.Name()
	defer os.Remove(archivePath)

	slog.InfoContext(ctx, "downloading Docker volume", "volume", source.name, "host", source.host)
	if err := runDockerVolumeScript(ctx, runner, source, exportDockerVolumeScript, nil, archive, stderr); err != nil {
		archive.Close()
		return fmt.Errorf("export source volume: %w", err)
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close temporary archive: %w", err)
	}

	archive, err = os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open temporary archive: %w", err)
	}
	defer archive.Close()

	slog.InfoContext(ctx, "uploading Docker volume", "volume", destination.name, "host", destination.host)
	if err := runDockerVolumeScript(ctx, runner, destination, importDockerVolumeScript, archive, nil, stderr); err != nil {
		return fmt.Errorf("import destination volume: %w", err)
	}

	slog.InfoContext(ctx, "Docker volume migration completed", "source", sourceValue, "destination", destinationValue)
	return nil
}

// parse `[user@]host:volumename`
func parseDockerVolumeLocation(value string) (dockerVolumeLocation, error) {
	separator := strings.LastIndex(value, ":")
	if separator <= 0 || separator == len(value)-1 {
		return dockerVolumeLocation{}, fmt.Errorf("invalid volume location %q: expected host:volumename", value)
	}

	return dockerVolumeLocation{
		host: value[:separator],
		name: value[separator+1:],
	}, nil
}

func runDockerVolumeScript(
	ctx context.Context,
	runner volumeCommandRunner,
	location dockerVolumeLocation,
	script string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) error {
	remoteCommand := "bash -c " + shellQuote(script) + " -- " + shellQuote(location.name)
	return runner(ctx, stdin, stdout, stderr, "ssh", "--", location.host, remoteCommand)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func nativeVolumeCommandRunner(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	name string,
	args ...string,
) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr

	return command.Run()
}

//go:embed export.sh
var exportDockerVolumeScript string

//go:embed import.sh
var importDockerVolumeScript string

type dockerVolumeLocation struct {
	host string
	name string
}

type volumeCommandRunner func(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	name string,
	args ...string,
) error
