package dev

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ironroot-dev",
		Short:        "Contributor helper CLI for IronRoot",
		Long:         "ironroot-dev is a contributor-only helper CLI for local IronRoot development workflows. It is not required for production deployments.",
		SilenceUsage: true,
	}
	cmd.AddCommand(devInitCommand())
	return cmd
}

func devInitCommand() *cobra.Command {
	opts := DevInitOptions{Output: ".localdev"}
	cmd := &cobra.Command{
		Use:   "dev-init",
		Short: "Initialize a local IronRoot development workspace",
		Long: "Initialize a neutral .localdev workspace for local IronRoot contributor workflows. " +
			"The command is self-contained, creates local data directories, and generates config from a template compiled into ironroot-dev.",
		Example: `  ironroot-dev dev-init
  ironroot-dev dev-init --repo /path/to/workspace
  ironroot-dev dev-init --output .localdev --dry-run
  ironroot-dev dev-init --force --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Stdout = cmd.OutOrStdout()
			opts.Stderr = cmd.ErrOrStderr()
			return RunDevInit(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "base directory for the local workspace; defaults to the current directory")
	cmd.Flags().StringVar(&opts.Output, "output", ".localdev", "local development workspace path; relative paths are resolved inside the base directory")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite generated config and helper files if they already exist")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "print actions without creating or modifying files")
	cmd.Flags().BoolVar(&opts.Verbose, "verbose", false, "print detailed path and file actions")
	return cmd
}
