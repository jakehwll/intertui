package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coder/serpent"
	"gopkg.in/yaml.v3"

	"intertui/internal/constants"
)

// InitOptions are flags for `intertui init`.
type InitOptions struct {
	Server  string
	Port    int64
	User    string
	Pass    string
	WS      bool
	TLS     bool
	URL     string
	Force   bool
}

// InitCmd returns the init subcommand.
func InitCmd() *serpent.Command {
	var opts InitOptions

	return &serpent.Command{
		Use:   "init",
		Short: "Create ~/.intertui with a config file.",
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
				Description: "Intercept username.",
			},
			{
				Name:        "pass",
				Flag:        "pass",
				Value:       serpent.StringOf(&opts.Pass),
				Description: "Intercept password.",
			},
			{
				Name:        "ws",
				Flag:        "ws",
				Value:       serpent.BoolOf(&opts.WS),
				Description: "Use WebSocket transport.",
			},
			{
				Name:        "tls",
				Flag:        "tls",
				Value:       serpent.BoolOf(&opts.TLS),
				Description: "Use wss:// instead of ws:// (with --ws).",
			},
			{
				Name:        "url",
				Flag:        "url",
				Value:       serpent.StringOf(&opts.URL),
				Description: "Full endpoint URL (overrides --server).",
			},
			{
				Name:        "force",
				Flag:        "force",
				Value:       serpent.BoolOf(&opts.Force),
				Description: "Overwrite an existing config file.",
			},
		},
		Handler: func(inv *serpent.Invocation) error {
			path, err := RunInit(opts)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(inv.Stdout, "Wrote %s\n", path)
			return err
		},
	}
}

// RunInit creates ~/.intertui and writes config.yaml.
func RunInit(opts InitOptions) (string, error) {
	if opts.Server == "" && opts.URL == "" {
		return "", fmt.Errorf("need --server or --url")
	}

	dir, err := Dir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, configName)
	if _, err := os.Stat(path); err == nil && !opts.Force {
		return "", fmt.Errorf("%s already exists (use --force to overwrite)", path)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	port := int(opts.Port)
	if port == 0 {
		port = constants.DEFAULT_PORT
	}

	file := configFile{
		Server: opts.Server,
		Port:   port,
		User:   opts.User,
		Pass:   opts.Pass,
		Token:  "",
		WS:     opts.WS,
		TLS:    opts.TLS,
		URL:    opts.URL,
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	byt, err := yaml.Marshal(file)
	if err != nil {
		return "", err
	}

	header := []byte("# intertui config — flags and env vars override these values.\n")
	if err := os.WriteFile(path, append(header, byt...), 0o600); err != nil {
		return "", err
	}

	return path, nil
}

type configFile struct {
	Server string `yaml:"server,omitempty"`
	Port   int    `yaml:"port,omitempty"`
	User   string `yaml:"user,omitempty"`
	Pass   string `yaml:"pass,omitempty"`
	Token  string `yaml:"token,omitempty"`
	WS     bool   `yaml:"ws,omitempty"`
	TLS    bool   `yaml:"tls,omitempty"`
	URL    string `yaml:"url,omitempty"`
}
