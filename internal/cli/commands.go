package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/mms/sleutel/internal/crypto"
	"github.com/mms/sleutel/internal/model"
	"github.com/mms/sleutel/internal/tui"
	"github.com/mms/sleutel/internal/vault"
)

// NewRootCmd builds and returns the root cobra command.
func NewRootCmd(defaultVaultPath string) *cobra.Command {
	var vaultPath string

	root := &cobra.Command{
		Use:   "sleutel",
		Short: "sleutel — local password manager",
		Long:  "sleutel is a local-first, encrypted password manager.",
	}
	root.PersistentFlags().StringVarP(&vaultPath, "vault", "v", defaultVaultPath, "path to vault file")

	root.AddCommand(
		newInitCmd(&vaultPath),
		newAddCmd(&vaultPath),
		newGetCmd(&vaultPath),
		newListCmd(&vaultPath),
		newEditCmd(&vaultPath),
		newDeleteCmd(&vaultPath),
		newSearchCmd(&vaultPath),
		newGenerateCmd(),
		newExportCmd(&vaultPath),
		newImportCmd(&vaultPath),
		newLockCmd(),
		newUnlockCmd(),
		newTUICmd(&vaultPath),
	)

	return root
}

// --- init ---

func newInitCmd(vaultPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a new vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			pw, err := readPasswordConfirm("Master password: ")
			if err != nil {
				return err
			}
			defer crypto.Zero(pw)

			v, err := vault.Create(*vaultPath, pw)
			if err != nil {
				return err
			}
			v.Close()
			fmt.Fprintf(cmd.OutOrStdout(), "Vault created at %s\n", *vaultPath)
			return nil
		},
	}
}

// --- add ---

func newAddCmd(vaultPath *string) *cobra.Command {
	var title, username, password, url, notes, tags string
	var generate bool
	var genLength int
	var genSymbols bool

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}

			pw, err := openVaultPassword()
			if err != nil {
				return err
			}
			defer crypto.Zero(pw)

			v, err := vault.Open(*vaultPath, pw)
			if err != nil {
				return err
			}
			defer v.Close()

			entryPw := password
			if generate {
				entryPw, err = vault.GeneratePassword(genLength, genSymbols)
				if err != nil {
					return fmt.Errorf("generate password: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Generated password: %s\n", entryPw)
			}

			e := model.Entry{
				Title:    title,
				Username: username,
				Password: entryPw,
				URL:      url,
				Notes:    notes,
			}
			if tags != "" {
				e.Tags = splitTags(tags)
			}

			added, err := v.Add(e)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added entry %s (%s)\n", added.ID, added.Title)
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "entry title (required)")
	cmd.Flags().StringVarP(&username, "username", "u", "", "username")
	cmd.Flags().StringVarP(&password, "password", "p", "", "password (use --generate to create one)")
	cmd.Flags().StringVar(&url, "url", "", "URL")
	cmd.Flags().StringVarP(&notes, "notes", "n", "", "notes")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	cmd.Flags().BoolVar(&generate, "generate", false, "generate a random password")
	cmd.Flags().IntVar(&genLength, "gen-length", 24, "generated password length")
	cmd.Flags().BoolVar(&genSymbols, "gen-symbols", true, "include symbols in generated password")

	return cmd
}

// --- get ---

func newGetCmd(vaultPath *string) *cobra.Command {
	var showPassword bool

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Show an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pw, err := openVaultPassword()
			if err != nil {
				return err
			}
			defer crypto.Zero(pw)

			v, err := vault.Open(*vaultPath, pw)
			if err != nil {
				return err
			}
			defer v.Close()

			e, err := v.Get(args[0])
			if err != nil {
				return err
			}
			printEntry(cmd.OutOrStdout(), e, showPassword)
			return nil
		},
	}
	cmd.Flags().BoolVar(&showPassword, "show", false, "reveal password in output")
	return cmd
}

// --- list ---

func newListCmd(vaultPath *string) *cobra.Command {
	var tag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			pw, err := openVaultPassword()
			if err != nil {
				return err
			}
			defer crypto.Zero(pw)

			v, err := vault.Open(*vaultPath, pw)
			if err != nil {
				return err
			}
			defer v.Close()

			entries := v.List()
			if tag != "" {
				entries = filterByTag(entries, tag)
			}
			printEntryTable(cmd.OutOrStdout(), entries)
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	return cmd
}

// --- edit ---

func newEditCmd(vaultPath *string) *cobra.Command {
	var title, username, password, url, notes, tags string

	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pw, err := openVaultPassword()
			if err != nil {
				return err
			}
			defer crypto.Zero(pw)

			v, err := vault.Open(*vaultPath, pw)
			if err != nil {
				return err
			}
			defer v.Close()

			patch := model.Entry{
				Title:    title,
				Username: username,
				Password: password,
				URL:      url,
				Notes:    notes,
			}
			if tags != "" {
				patch.Tags = splitTags(tags)
			}

			updated, err := v.Edit(args[0], patch)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated entry %s (%s)\n", updated.ID, updated.Title)
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "new title")
	cmd.Flags().StringVarP(&username, "username", "u", "", "new username")
	cmd.Flags().StringVarP(&password, "password", "p", "", "new password")
	cmd.Flags().StringVar(&url, "url", "", "new URL")
	cmd.Flags().StringVarP(&notes, "notes", "n", "", "new notes")
	cmd.Flags().StringVar(&tags, "tags", "", "new comma-separated tags")

	return cmd
}

// --- delete ---

func newDeleteCmd(vaultPath *string) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force && !confirm(fmt.Sprintf("Delete entry %s?", args[0])) {
				fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
				return nil
			}

			pw, err := openVaultPassword()
			if err != nil {
				return err
			}
			defer crypto.Zero(pw)

			v, err := vault.Open(*vaultPath, pw)
			if err != nil {
				return err
			}
			defer v.Close()

			if err := v.Delete(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted entry %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation")
	return cmd
}

// --- search ---

func newSearchCmd(vaultPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search entries by title, URL, notes, or tags",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pw, err := openVaultPassword()
			if err != nil {
				return err
			}
			defer crypto.Zero(pw)

			v, err := vault.Open(*vaultPath, pw)
			if err != nil {
				return err
			}
			defer v.Close()

			results := v.Search(args[0])
			printEntryTable(cmd.OutOrStdout(), results)
			return nil
		},
	}
}

// --- generate ---

func newGenerateCmd() *cobra.Command {
	var length int
	var symbols bool

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a random password",
		RunE: func(cmd *cobra.Command, args []string) error {
			pw, err := vault.GeneratePassword(length, symbols)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), pw)
			return nil
		},
	}
	cmd.Flags().IntVarP(&length, "length", "l", 24, "password length")
	cmd.Flags().BoolVarP(&symbols, "symbols", "s", true, "include symbols")
	return cmd
}

// --- export ---

func newExportCmd(vaultPath *string) *cobra.Command {
	var outFile string
	var confirmed bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export vault entries to plaintext JSON",
		Long: `Export all vault entries to a plaintext JSON file.

WARNING: the export file is NOT encrypted. Store it securely and delete it when done.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmed {
				fmt.Fprintln(os.Stderr, "WARNING: export writes passwords in plaintext.")
				if !confirm("I understand. Proceed with export?") {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			pw, err := openVaultPassword()
			if err != nil {
				return err
			}
			defer crypto.Zero(pw)

			v, err := vault.Open(*vaultPath, pw)
			if err != nil {
				return err
			}
			defer v.Close()

			data, err := json.MarshalIndent(v.Entries(), "", "  ")
			if err != nil {
				return fmt.Errorf("marshal: %w", err)
			}

			if outFile == "" || outFile == "-" {
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
			} else {
				if err := os.WriteFile(outFile, data, 0600); err != nil {
					return fmt.Errorf("write file: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Exported %d entries to %s\n", len(v.Entries()), outFile)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&outFile, "file", "f", "-", "output file path (- for stdout)")
	cmd.Flags().BoolVar(&confirmed, "yes", false, "skip confirmation prompt")
	return cmd
}

// --- import ---

func newImportCmd(vaultPath *string) *cobra.Command {
	var inFile string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import entries from a JSON export file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inFile == "" {
				return fmt.Errorf("--file is required")
			}

			data, err := os.ReadFile(inFile)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			var entries []model.Entry
			if err := json.Unmarshal(data, &entries); err != nil {
				return fmt.Errorf("parse JSON: %w", err)
			}

			pw, err := openVaultPassword()
			if err != nil {
				return err
			}
			defer crypto.Zero(pw)

			v, err := vault.Open(*vaultPath, pw)
			if err != nil {
				return err
			}
			defer v.Close()

			if err := v.ImportEntries(entries); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Imported %d entries\n", len(entries))
			return nil
		},
	}
	cmd.Flags().StringVarP(&inFile, "file", "f", "", "JSON file to import (required)")
	return cmd
}

// --- tui ---

func newTUICmd(vaultPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(*vaultPath); err != nil {
				return fmt.Errorf("vault not found at %s — run 'sleutel init' first", *vaultPath)
			}
			p := tea.NewProgram(
				tui.NewApp(*vaultPath),
				tea.WithAltScreen(),
			)
			_, err := p.Run()
			return err
		},
	}
}

// --- lock / unlock (phase 1 placeholders) ---

func newLockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lock",
		Short: "Lock the vault (session management, phase 2)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "Session management is not implemented in phase 1. The vault is always locked at rest.")
			return nil
		},
	}
}

func newUnlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlock",
		Short: "Unlock the vault (session management, phase 2)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "Session management is not implemented in phase 1. Use any command — you will be prompted for your master password.")
			return nil
		},
	}
}

// --- helpers ---

func openVaultPassword() ([]byte, error) {
	return readPassword("Master password: ")
}

func printEntry(w interface{ Write([]byte) (int, error) }, e model.Entry, showPw bool) {
	fmt.Fprintf(w, "ID:       %s\n", e.ID)
	fmt.Fprintf(w, "Title:    %s\n", e.Title)
	if e.Username != "" {
		fmt.Fprintf(w, "Username: %s\n", e.Username)
	}
	if showPw {
		fmt.Fprintf(w, "Password: %s\n", e.Password)
	} else if e.Password != "" {
		fmt.Fprintf(w, "Password: ********\n")
	}
	if e.URL != "" {
		fmt.Fprintf(w, "URL:      %s\n", e.URL)
	}
	if e.Notes != "" {
		fmt.Fprintf(w, "Notes:    %s\n", e.Notes)
	}
	if len(e.Tags) > 0 {
		fmt.Fprintf(w, "Tags:     %s\n", strings.Join(e.Tags, ", "))
	}
	fmt.Fprintf(w, "Created:  %s\n", e.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated:  %s\n", e.UpdatedAt.Format("2006-01-02 15:04:05"))
}

func printEntryTable(out interface{ Write([]byte) (int, error) }, entries []model.Entry) {
	if len(entries) == 0 {
		fmt.Fprintln(out, "No entries found.")
		return
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tUSERNAME\tURL\tTAGS")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.ID[:8]+"...", e.Title, e.Username, e.URL, strings.Join(e.Tags, ","))
	}
	w.Flush()
}

func splitTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func filterByTag(entries []model.Entry, tag string) []model.Entry {
	tag = strings.ToLower(tag)
	var out []model.Entry
	for _, e := range entries {
		for _, t := range e.Tags {
			if strings.ToLower(t) == tag {
				out = append(out, e)
				break
			}
		}
	}
	return out
}
