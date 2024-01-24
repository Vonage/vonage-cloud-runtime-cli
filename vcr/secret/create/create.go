package create

import (
	"context"
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
	"vcr-cli/pkg/api"
	"vcr-cli/pkg/cmdutil"
	"vcr-cli/pkg/config"
)

type Options struct {
	cmdutil.Factory

	Name       string
	Value      string
	SecretFile string
}

func NewCmdSecretCreate(f cmdutil.Factory) *cobra.Command {
	opts := Options{
		Factory: f,
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: `Create a new secret`,
		Long: heredoc.Doc(`Create a new secret.

			The secrets can be loaded into your deployed applications as environment variables.
			
			Each secret will be capitalised and prefixed with NERU_SECRET_ when being injected in as an environment variable.
		`),
		Example: heredoc.Doc(`
			$  vcr secret create --name <name> --value <value>
		
			$  vcr secret create --name <name> --file <path/to/file>
		`),
		Args:    cobra.MaximumNArgs(0),
		Aliases: []string{"add"},

		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithDeadline(context.Background(), opts.Deadline())
			defer cancel()

			return runCreate(ctx, &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "The name of the secret")
	cmd.Flags().StringVarP(&opts.Value, "value", "v", "", "The value of the secret")
	cmd.Flags().StringVarP(&opts.SecretFile, "filename", "f", "", "The path to the file containing the secret")

	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runCreate(ctx context.Context, opts *Options) error {
	io := opts.IOStreams()
	c := opts.IOStreams().ColorScheme()

	_, err := config.ValidateSecretName(opts.Name)
	if err != nil {
		return fmt.Errorf("invalid secret name: %w", err)
	}

	secret, err := config.GetSecretFromInputs(opts.IOStreams(), opts.Name, opts.Value, opts.SecretFile)
	if err != nil {
		return fmt.Errorf("can't read secret's value: %w", err)
	}

	spinner := cmdutil.DisplaySpinnerMessageWithHandle(fmt.Sprintf("Creating secret %q...", opts.Name))
	err = opts.DeploymentClient().CreateSecret(ctx, secret)
	spinner.Stop()
	switch {
	case errors.Is(err, api.ErrAlreadyExists):
		return fmt.Errorf("secret %q already exists", opts.Name)
	case err != nil:
		return fmt.Errorf("failed to create secret: %w", err)
	}

	fmt.Fprintf(io.Out, "%s Secret %q created\n", c.SuccessIcon(), opts.Name)
	return nil
}
