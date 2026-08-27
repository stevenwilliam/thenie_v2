package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/stevenwilliam/thenie_v2/server/internal/adapter/postgres"
	"github.com/stevenwilliam/thenie_v2/server/internal/app/admin"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/config"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/database"
)

// runUser is how the FIRST account gets created. There is no sign-up page and
// no default password, so without this subcommand a freshly migrated database
// has an admin UI nobody can log into.
func runUser(ctx context.Context, cfg *config.Config, log *slogLogger, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, `thenied user — manage admin accounts

  thenied user list
  thenied user create --email a@b.com --name "Nama" [--roles owner,editor]
  thenied user password --email a@b.com
  thenied user roles --email a@b.com --roles manager
  thenied user activate|deactivate --email a@b.com

The password is prompted for, never passed on the command line: an argument
lands in shell history and in the process list where any other user can read it.
`)
		return errors.New("no user subcommand given")
	}

	db, err := database.Open(ctx, database.Options{URL: cfg.DatabaseURL}, log)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	repo := postgres.NewAdminRepo(db)
	svc := admin.NewService(repo, cfg.AdminToken)

	flags := parseFlags(args[1:])
	email := admin.NormalizeEmail(flags["email"])

	switch args[0] {
	case "list":
		users, err := svc.ListUsers(ctx)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			fmt.Println("no admin accounts yet — create one with: thenied user create --email ... --name ...")
			return nil
		}
		fmt.Printf("%-32s %-24s %-8s %s\n", "EMAIL", "NAME", "ACTIVE", "ROLES")
		for _, u := range users {
			fmt.Printf("%-32s %-24s %-8v %s\n", u.Email, u.Name, u.IsActive, strings.Join(u.Roles, ","))
		}
		return nil

	case "create":
		if email == "" || flags["name"] == "" {
			return errors.New("user create: --email and --name are required")
		}
		roles := splitCSV(flags["roles"])
		if len(roles) == 0 {
			roles = []string{"owner"}
		}
		pw, err := promptPassword("Password for " + email + ": ")
		if err != nil {
			return err
		}
		again, err := promptPassword("Repeat: ")
		if err != nil {
			return err
		}
		if pw != again {
			return errors.New("passwords do not match")
		}
		id, err := svc.CreateUser(ctx, email, flags["name"], pw, roles)
		if err != nil {
			return err
		}
		_ = repo.WriteAudit(ctx, "", "cli", "user.create", email, map[string]any{"roles": roles}, "cli")
		fmt.Printf("created %s (%s) with role(s) %s\n", email, id, strings.Join(roles, ","))
		return nil

	case "password":
		if email == "" {
			return errors.New("user password: --email is required")
		}
		u, err := repo.FindUserByEmail(ctx, email)
		if err != nil {
			return err
		}
		if u == nil {
			return fmt.Errorf("no account %s", email)
		}
		pw, err := promptPassword("New password for " + email + ": ")
		if err != nil {
			return err
		}
		again, err := promptPassword("Repeat: ")
		if err != nil {
			return err
		}
		if pw != again {
			return errors.New("passwords do not match")
		}
		if err := svc.SetPassword(ctx, u.ID, pw); err != nil {
			return err
		}
		_ = repo.WriteAudit(ctx, u.ID, "cli", "user.password", email, nil, "cli")
		fmt.Printf("password changed for %s; existing sessions revoked\n", email)
		return nil

	case "roles":
		if email == "" || flags["roles"] == "" {
			return errors.New("user roles: --email and --roles are required")
		}
		u, err := repo.FindUserByEmail(ctx, email)
		if err != nil {
			return err
		}
		if u == nil {
			return fmt.Errorf("no account %s", email)
		}
		if err := svc.SetUserRoles(ctx, nil, u.ID, splitCSV(flags["roles"])); err != nil {
			return err
		}
		fmt.Printf("%s now holds role(s) %s\n", email, flags["roles"])
		return nil

	case "activate", "deactivate":
		if email == "" {
			return errors.New("--email is required")
		}
		u, err := repo.FindUserByEmail(ctx, email)
		if err != nil {
			return err
		}
		if u == nil {
			return fmt.Errorf("no account %s", email)
		}
		active := args[0] == "activate"
		if err := svc.UpdateUser(ctx, nil, u.ID, u.Name, active); err != nil {
			return err
		}
		fmt.Printf("%s is now active=%v\n", email, active)
		return nil

	default:
		return fmt.Errorf("user: unknown subcommand %q", args[0])
	}
}

// stdinReader is created ONCE. A bufio.Reader reads ahead, so building a fresh
// one per prompt throws away whatever it buffered past the first newline — which
// made the second "Repeat:" prompt hit EOF whenever input was piped rather than
// typed. Found the first time this command was run non-interactively.
var stdinReader = bufio.NewReader(os.Stdin)

// promptPassword reads without echoing. Falling back to a visible read when
// stdin is not a terminal keeps the command usable in a pipe, and says so.
func promptPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("reading password from stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func parseFlags(args []string) map[string]string {
	out := map[string]string{}
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") {
			continue
		}
		key := strings.TrimPrefix(args[i], "--")
		if eq := strings.Index(key, "="); eq >= 0 {
			out[key[:eq]] = key[eq+1:]
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			out[key] = args[i+1]
			i++
		} else {
			out[key] = "true"
		}
	}
	return out
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
