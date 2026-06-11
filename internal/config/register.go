package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/coder/serpent"

	"intertui/internal/constants"
	"intertui/internal/intercept"
)

// RegisterConfigConflict is returned when registration succeeds but config.yaml exists.
type RegisterConfigConflict struct {
	Player string
	Exist  *ConfigExistsError
}

func (e *RegisterConfigConflict) Error() string {
	return e.Exist.Error()
}

func (e *RegisterConfigConflict) Unwrap() error {
	return e.Exist
}

// RegisterOptions are flags for `intertui register`.
type RegisterOptions struct {
	Server string
	Port   int64
	User   string
	Pass   string
	Force  bool
}

// RegisterCmd returns the register subcommand.
func RegisterCmd() *serpent.Command {
	var opts RegisterOptions

	return &serpent.Command{
		Use:   "register",
		Short: "Create an Intercept account and write ~/.intertui/config.yaml.",
		Options: serpent.OptionSet{
			{
				Name:        "server",
				Flag:        "server",
				Value:       serpent.StringOf(&opts.Server),
				Description: "Game server host or IP.",
			},
			{
				Name:        "port",
				Flag:        "port",
				Default:     fmt.Sprintf("%d", constants.DEFAULT_PORT),
				Value:       serpent.Int64Of(&opts.Port),
				Description: fmt.Sprintf("Server port (default %d).", constants.DEFAULT_PORT),
			},
			{
				Name:        "user",
				Flag:        "user",
				Value:       serpent.StringOf(&opts.User),
				Description: "New Intercept username.",
			},
			{
				Name:        "pass",
				Flag:        "pass",
				Value:       serpent.StringOf(&opts.Pass),
				Description: "New Intercept password.",
			},
			{
				Name:        "force",
				Flag:        "force",
				Value:       serpent.BoolOf(&opts.Force),
				Description: "Overwrite an existing config file.",
			},
		},
		Handler: func(inv *serpent.Invocation) error {
			player, path, err := RunRegister(inv.Context(), opts)
			if err != nil {
				var conflict *RegisterConfigConflict
				if errors.As(err, &conflict) {
					_, _ = fmt.Fprintf(inv.Stdout, "Registered as %s\n\n%s\n\n%s", conflict.Player, conflict.Exist, conflict.Exist.YAML)
				}
				return err
			}
			_, err = fmt.Fprintf(inv.Stdout, "Registered as %s\nWrote %s\n", player, path)
			return err
		},
	}
}

// RunRegister creates an account on the game server and writes config.yaml.
func RunRegister(ctx context.Context, opts RegisterOptions) (player, path string, err error) {
	if opts.Server == "" {
		return "", "", fmt.Errorf("need --server")
	}
	if opts.User == "" || opts.Pass == "" {
		return "", "", fmt.Errorf("need --user and --pass")
	}

	port := int(opts.Port)
	if port == 0 {
		port = constants.DEFAULT_PORT
	}

	cfg := Config{Server: opts.Server, Port: port}
	player, err = intercept.Register(ctx, cfg.ResolveAddr(), opts.User, opts.Pass)
	if err != nil {
		return "", "", err
	}

	initOpts := InitOptions{
		Server: opts.Server,
		Port:   int64(port),
		User:   opts.User,
		Pass:   opts.Pass,
		Force:  opts.Force,
	}
	path, err = RunInit(initOpts)
	if err != nil {
		var exist *ConfigExistsError
		if errors.As(err, &exist) {
			return "", "", &RegisterConfigConflict{Player: player, Exist: exist}
		}
		return "", "", err
	}

	return player, path, nil
}
