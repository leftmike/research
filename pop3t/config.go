package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/knadh/go-pop3"
)

type pop3Config struct {
	Host     string `hcl:"host,optional"`
	Port     int    `hcl:"port,optional"`
	User     string `hcl:"user,optional"`
	Password string `hcl:"password,optional"`
	NoTLS    bool   `hcl:"no_tls,optional"`
}

type smtpConfig struct {
	Host     string `hcl:"host,optional"`
	Port     int    `hcl:"port,optional"`
	User     string `hcl:"user,optional"`
	Password string `hcl:"password,optional"`
	NoTLS    bool   `hcl:"no_tls,optional"`
}

type config struct {
	Host     string     `hcl:"host,optional"`
	Port     int        `hcl:"port,optional"`
	User     string     `hcl:"user,optional"`
	Password string     `hcl:"password,optional"`
	POP3     pop3Config `hcl:"pop3,optional"`
	SMTP     smtpConfig `hcl:"smtp,optional"`
}

func (cfg *config) pop3Host() string {
	if cfg.POP3.Host != "" {
		return cfg.POP3.Host
	}
	return cfg.Host
}

func (cfg *config) pop3Port() int {
	if cfg.POP3.Port != 0 {
		return cfg.POP3.Port
	}
	return cfg.Port
}
func (cfg *config) pop3User() string {
	if cfg.POP3.User != "" {
		return cfg.POP3.User
	}
	return cfg.User
}
func (cfg *config) pop3Password() string {
	if cfg.POP3.Password != "" {
		return cfg.POP3.Password
	}
	return cfg.Password
}

func configFilenames() []string {
	if runtime.GOOS == "windows" {
		return []string{"~/pop3t/pop3t.hcl", "~/pop3t.hcl", "./pop3t.hcl"}
	}

	return []string{"~/.pop3t/pop3t.hcl", "~/.pop3t.hcl", "./pop3t.hcl"}
}

func readConfig(filenames []string) (*config, error) {
	for _, name := range filenames {
		buf, err := os.ReadFile(name)
		if err == nil {
			var cfg config
			err := hclsimple.Decode(name, buf, nil, &cfg)
			return &cfg, err
		}
	}

	return nil, nil
}

func loadConfig(fs *flag.FlagSet, args []string) (*config, []string) {
	var configFile string

	fs.StringVar(&configFile, "config", "", "HCL config file")
	fs.String("pop3-host", "", "POP3 server host")
	fs.Int("pop3-port", 0, "POP3 server port")
	fs.String("pop3-user", "", "POP3 username")
	fs.String("pop3-password", "", "POP3 password")
	fs.Bool("pop3-no-tls", false, "POP3 disable TLS")
	fs.Parse(args)

	var cfg *config
	if configFile != "" {
		var err error
		cfg, err = readConfig([]string{configFile})
		if err != nil {
			fatal(err)
		} else if cfg == nil {
			fatal(fmt.Errorf("config file not found: %s", configFile))
		}
	} else {
		var err error
		cfg, err = readConfig(configFilenames())
		if err != nil {
			fatal(err)
		} else if cfg == nil {
			cfg = &config{}
		}
	}

	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "pop3-host":
			cfg.POP3.Host = f.Value.String()
		case "pop3-port":
			cfg.POP3.Port, _ = strconv.Atoi(f.Value.String())
		case "pop3-user":
			cfg.POP3.User = f.Value.String()
		case "pop3-password":
			cfg.POP3.Password = f.Value.String()
		case "pop3-no-tls":
			cfg.POP3.NoTLS = (f.Value.String() == "true")
		}
	})

	// XXX: if verbose
	// fmt.Printf("%s:%d %s tls:%v\n", cfg.Host, cfg.Port, cfg.User, !cfg.NoTLS)
	return cfg, fs.Args()
}

func (cfg *config) newConn() *pop3.Conn {
	host := cfg.pop3Host()
	user := cfg.pop3User()
	password := cfg.pop3Password()
	if host == "" || user == "" || password == "" {
		fatal(errors.New("host, user, and password are required via config or flag"))
	}

	port := cfg.pop3Port()
	if port == 0 {
		if cfg.POP3.NoTLS {
			port = 110
		} else {
			port = 995
		}
	}

	conn, err := pop3.New(pop3.Opt{
		Host:       host,
		Port:       port,
		TLSEnabled: !cfg.POP3.NoTLS,
	}).NewConn()
	if err != nil {
		fatal(err)
	}

	err = conn.Auth(user, password)
	if err != nil && !strings.Contains(user, "@") {
		err = conn.Auth(fmt.Sprintf("%s@%s", user, host), password)
	}
	if err != nil {
		fatal(err)
	}

	return conn
}
