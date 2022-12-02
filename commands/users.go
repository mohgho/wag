package commands

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/NHAS/wag/control/wagctl"
)

type users struct {
	fs *flag.FlagSet

	username string
	action   string
}

func Users() *users {
	gc := &users{
		fs: flag.NewFlagSet("users", flag.ContinueOnError),
	}

	gc.fs.StringVar(&gc.username, "username", "", "Username to act upon")

	gc.fs.Bool("del", false, "Delete user and all associated devices")
	gc.fs.Bool("list", false, "List users, if '-username' supply will filter by user")

	gc.fs.Bool("lock", false, "Locked account, disables all MFA on all devices and deauthenticates all active sessions")
	gc.fs.Bool("unlock", false, "Unlock a locked account")

	gc.fs.Bool("reset-mfa", false, "Reset MFA details, invalids all session and set MFA to be shown")

	return gc
}

func (g *users) FlagSet() *flag.FlagSet {
	return g.fs
}

func (g *users) Name() string {

	return g.fs.Name()
}

func (g *users) PrintUsage() {
	g.fs.Usage()
}

func (g *users) Check() error {
	g.fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "lock", "unlock", "del", "list", "reset-mfa":
			g.action = strings.ToLower(f.Name)
		}
	})

	switch g.action {
	case "del", "unlock", "lock", "reset-mfa":
		if g.username == "" {
			return errors.New("address must be supplied")
		}
	case "list":
	default:
		return errors.New("Unknown flag: " + g.action)
	}

	return nil
}

func (g *users) Run() error {
	switch g.action {
	case "del":

		err := wagctl.DeleteUser(g.username)
		if err != nil {
			return err
		}

		fmt.Println("OK")
	case "list":

		users, err := wagctl.ListUsers(g.username)
		if err != nil {
			return err
		}

		fmt.Println("username,locked,enforcingmfa")
		for _, user := range users {
			fmt.Printf("%s,%t,%t\n", user.Username, user.Locked, user.Enforcing)
		}
	case "lock":

		err := wagctl.LockUser(g.username)
		if err != nil {
			return err
		}

		fmt.Println("OK")

	case "unlock":

		err := wagctl.UnlockUser(g.username)
		if err != nil {
			return err
		}

		fmt.Println("OK")
	case "reset-mfa":
		err := wagctl.ResetUserMFA(g.username)
		if err != nil {
			return err
		}
		fmt.Println("OK")
	}

	return nil
}
