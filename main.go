package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/bamorim/portpls/internal/app"
)

var (
	version string
	commit  string
	date    string
)

func getVersion() string {
	if version == "" {
		return "dev"
	}
	if commit != "" {
		return fmt.Sprintf("%s (%s)", version, commit[:7])
	}
	return version
}

func main() {
	appCLI := &cli.App{
		Name:    "portpls",
		Usage:   "Port allocation CLI",
		Version: getVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Usage: "Path to config file"},
			&cli.StringFlag{Name: "allocations", Usage: "Path to allocations file"},
			&cli.StringFlag{Name: "directory", Usage: "Override current directory"},
			&cli.BoolFlag{Name: "verbose", Usage: "Enable debug output"},
		},
		Commands: []*cli.Command{
			getCommand(),
			listCommand(),
			lockCommand(),
			unlockCommand(),
			forgetCommand(),
			scanCommand(),
			configCommand(),
		},
	}

	if err := appCLI.Run(os.Args); err != nil {
		if exitErr, ok := err.(cli.ExitCoder); ok {
			if msg := exitErr.Error(); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
			}
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a free port for the current directory",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "main", Usage: "Named allocation"},
		},
		Action: func(c *cli.Context) error {
			portNum, err := app.GetPort(optionsFromContext(c), c.String("name"))
			if err != nil {
				return exitForError(err)
			}
			fmt.Fprintln(os.Stdout, portNum)
			return nil
		},
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all port allocations",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "table", Usage: "Output format: table, json"},
			&cli.StringFlag{Name: "directory", Usage: "Filter by directory"},
		},
		Action: func(c *cli.Context) error {
			entries, err := app.ListAllocations(optionsFromContext(c))
			if err != nil {
				return exitForError(err)
			}
			format := strings.ToLower(c.String("format"))
			switch format {
			case "json":
				return outputJSON(entries)
			case "table":
				return outputTable(entries)
			default:
				return cli.Exit("unknown format", 1)
			}
		},
	}
}

func lockCommand() *cli.Command {
	return &cli.Command{
		Name:  "lock",
		Usage: "Lock a port to prevent reallocation",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "main", Usage: "Named allocation"},
		},
		Action: func(c *cli.Context) error {
			portNum, err := app.LockPort(optionsFromContext(c), c.String("name"))
			if err != nil {
				return exitForError(err)
			}
			fmt.Fprintf(os.Stdout, "Locked port %d\n", portNum)
			return nil
		},
	}
}

func unlockCommand() *cli.Command {
	return &cli.Command{
		Name:  "unlock",
		Usage: "Unlock a port",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "main", Usage: "Named allocation"},
			&cli.StringFlag{Name: "directory", Usage: "Override directory"},
		},
		Action: func(c *cli.Context) error {
			portNum, err := app.UnlockPort(optionsFromContext(c), c.String("name"))
			if err != nil {
				return exitForError(err)
			}
			fmt.Fprintf(os.Stdout, "Unlocked port %d\n", portNum)
			return nil
		},
	}
}

func forgetCommand() *cli.Command {
	return &cli.Command{
		Name:  "forget",
		Usage: "Remove port allocations",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "main", Usage: "Named allocation"},
			&cli.BoolFlag{Name: "all", Usage: "Remove all allocations for current directory"},
			&cli.BoolFlag{Name: "all-directories", Usage: "Remove all allocations for all directories"},
			&cli.StringFlag{Name: "directory", Usage: "Override directory"},
		},
		Action: func(c *cli.Context) error {
			result, err := app.Forget(
				optionsFromContext(c),
				c.String("name"),
				c.IsSet("name"),
				c.Bool("all"),
				c.Bool("all-directories"),
				confirmAll,
			)
			if err != nil {
				return exitForError(err)
			}
			if result.Message != "" {
				fmt.Fprintln(os.Stdout, result.Message)
			}
			return nil
		},
	}
}

func scanCommand() *cli.Command {
	return &cli.Command{
		Name:  "scan",
		Usage: "Scan port range and record busy ports",
		Action: func(c *cli.Context) error {
			result, err := app.Scan(optionsFromContext(c))
			if err != nil {
				return exitForError(err)
			}
			fmt.Fprintf(os.Stdout, "Scanning ports %d-%d...\n", result.Start, result.End)
			for _, line := range result.Lines {
				fmt.Fprintln(os.Stdout, line)
			}
			fmt.Fprintf(os.Stdout, "Recorded %d new allocation(s)\n", result.Added)
			return nil
		},
	}
}

func configCommand() *cli.Command {
	return &cli.Command{
		Name:      "config",
		Usage:     "Show or modify configuration",
		ArgsUsage: "[KEY] [VALUE]",
		Action: func(c *cli.Context) error {
			if c.Args().Len() == 0 {
				lines, err := app.ConfigShow(optionsFromContext(c))
				if err != nil {
					return exitForError(err)
				}
				for _, line := range lines {
					fmt.Fprintln(os.Stdout, line)
				}
				return nil
			}
			key := c.Args().Get(0)
			if c.Args().Len() == 1 {
				value, err := app.ConfigGet(optionsFromContext(c), key)
				if err != nil {
					return exitForError(err)
				}
				fmt.Fprintln(os.Stdout, value)
				return nil
			}
			value := c.Args().Get(1)
			line, err := app.ConfigSet(optionsFromContext(c), key, value)
			if err != nil {
				return exitForError(err)
			}
			fmt.Fprintln(os.Stdout, line)
			return nil
		},
	}
}

func optionsFromContext(c *cli.Context) app.Options {
	directory := c.String("directory")
	directorySet := c.IsSet("directory")
	parentDirectory := ""
	parentDirectorySet := false
	if !directorySet {
		parentDirectory, parentDirectorySet = parentDirectoryFromContext(c)
	}
	return app.Options{
		ConfigPath:         c.String("config"),
		AllocationsPath:    c.String("allocations"),
		Directory:          directory,
		DirectorySet:       directorySet,
		ParentDirectory:    parentDirectory,
		ParentDirectorySet: parentDirectorySet,
		Verbose:            c.Bool("verbose"),
	}
}

func parentDirectoryFromContext(c *cli.Context) (string, bool) {
	lineage := c.Lineage()
	if len(lineage) == 0 {
		return "", false
	}
	root := lineage[len(lineage)-1]
	if root == nil || root == c {
		return "", false
	}
	if root.IsSet("directory") {
		return root.String("directory"), true
	}
	return "", false
}

func exitForError(err error) error {
	if err == nil {
		return nil
	}
	var codeErr app.CodeError
	if errors.As(err, &codeErr) {
		return cli.Exit(codeErr.Error(), codeErr.Code)
	}
	return cli.Exit(err.Error(), 2)
}

func outputJSON(entries []app.AllocationEntry) error {
	payload, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(payload))
	return nil
}

func outputTable(entries []app.AllocationEntry) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "PORT\tDIRECTORY\tNAME\tSTATUS\tLOCKED\tASSIGNED\tLAST_USED")
	for _, entry := range entries {
		locked := "no"
		if entry.Locked {
			locked = "yes"
		}
		fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			entry.Port,
			shortenHome(entry.Directory),
			entry.Name,
			entry.Status,
			locked,
			formatTimestamp(entry.AssignedAt),
			formatTimestamp(entry.LastUsedAt),
		)
	}
	return writer.Flush()
}

func formatTimestamp(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04")
}

func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		remainder := strings.TrimPrefix(path, home)
		if remainder == "" {
			return "~"
		}
		return filepath.Join("~", remainder)
	}
	return path
}

func confirmAll() bool {
	fmt.Fprint(os.Stdout, "This will remove ALL port allocations. Continue? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
