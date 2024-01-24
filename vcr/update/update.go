package update

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/blang/semver"
	"github.com/inconshreveable/go-update"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
	"vcr-cli/pkg/api"
	"vcr-cli/pkg/cmdutil"
)

type Options struct {
	cmdutil.Factory

	forceUpdate bool
}

func NewCmdUpdate(f cmdutil.Factory, version, buildDate, commit string) *cobra.Command {
	opts := Options{
		Factory: f,
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: `Show and update VCR CLI version`,
		Long: heredoc.Doc(`Show VCR CLI version. 

			If current version is not the latest, the option to update will be provided.
		`),
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithDeadline(context.Background(), opts.Deadline())
			defer cancel()
			fmt.Fprint(f.IOStreams().Out, cmd.Root().Annotations["versionInfo"])
			return runUpdate(ctx, &opts, version, buildDate, commit)
		},
	}

	cmd.Flags().BoolVarP(&opts.forceUpdate, "force", "f", false, "force update and skip prompt if new update exists")

	return cmd
}

func runUpdate(ctx context.Context, opts *Options, version, buildDate, commit string) error {
	io := opts.IOStreams()
	c := opts.IOStreams().ColorScheme()

	current, err := GetCurrentVersion(version)
	if err != nil {
		return fmt.Errorf("current update is invalid: %s", err)
	}

	spinner := cmdutil.DisplaySpinnerMessageWithHandle(" Checking for update...")
	release, err := opts.ReleaseClient().GetLatestRelease(ctx)
	spinner.Stop()
	if err != nil {
		return fmt.Errorf("failed to get assets: %w", err)
	}

	latest, err := GetLatestVersion(release)
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	if latest.LTE(current) {
		if latest.EQ(current) {
			fmt.Fprintf(io.Out, "%s You are using the latest version of vcr-cli (%s)\n", c.SuccessIcon(), current.String())
		}
		if current.GT(latest) {
			fmt.Fprintf(io.Out, "%s Current version (%s) is newer than the latest version (%s) !\n", c.SuccessIcon(), current.String(), latest.String())
		}
		return nil
	}

	latestVersion := latest.String()

	if io.CanPrompt() && !opts.forceUpdate {
		if !opts.Survey().AskYesNo(fmt.Sprintf("Are you sure you want to update to %s ?", latestVersion)) {
			fmt.Fprintf(io.ErrOut, "%s Update aborted\n", c.WarningIcon())
			return nil
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	spinner = cmdutil.DisplaySpinnerMessageWithHandle(fmt.Sprintf(" Updating CLI to latest version - v%s...", latestVersion))
	err = updateByAsset(ctx, opts, release, exePath)
	spinner.Stop()
	if err != nil {
		return err
	}

	fmt.Fprintf(io.Out, "%s Successfully updated to update %s\n", c.SuccessIcon(), latestVersion)

	return nil
}

func Format(version, buildDate, commit string) string {
	if version == "dev" {
		version = "0.0.1"
	}
	if buildDate != "" {
		version = fmt.Sprintf("%s (commit:%s, date:%s)", version, commit, buildDate)
	}
	return fmt.Sprintf("vcr-cli version %s\n", version)
}

func GetCurrentVersion(v string) (semver.Version, error) {
	version := strings.TrimPrefix(v, "v")
	if version == "dev" {
		version = "0.0.1"
	}
	current, err := semver.Parse(version)
	if err != nil {
		return semver.Version{}, err
	}
	return current, nil
}

func GetLatestVersion(release api.Release) (semver.Version, error) {
	releaseVersion := strings.TrimPrefix(release.TagName, "v")
	parsedVersion, err := semver.Parse(releaseVersion)
	if err != nil {
		return semver.Version{}, fmt.Errorf("invalid update found: %s", err)
	}

	return parsedVersion, nil
}

func updateByAsset(ctx context.Context, opts *Options, release api.Release, exePath string) error {
	latestAssetURL, err := getDownloadURL(release)
	if err != nil {
		return fmt.Errorf("failed to get download url: %w", err)
	}

	asset, err := opts.ReleaseClient().GetAsset(ctx, latestAssetURL)
	if err != nil {
		return fmt.Errorf("failed to get release asset: %w", err)
	}

	_, baseName := filepath.Split(exePath)
	cmd := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	binary, err := selfupdate.UncompressCommand(bytes.NewReader(asset), latestAssetURL, cmd)
	if err != nil {
		return fmt.Errorf("failed to uncompress command: %w", err)
	}

	err = update.Apply(binary, update.Options{
		TargetPath: exePath,
	})
	if err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}
	return nil
}

func getDownloadURL(release api.Release) (string, error) {
	for _, asset := range release.Assets {
		if asset.Name == fmt.Sprintf("vcr_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH) {
			if asset.BrowserDownloadURL == "" {
				return "", fmt.Errorf("download url not found for %s %s", runtime.GOOS, runtime.GOARCH)
			}
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no asset found for %s %s", runtime.GOOS, runtime.GOARCH)
}
