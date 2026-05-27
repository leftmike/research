package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/knadh/go-pop3"
)

type config struct {
	Host     string `hcl:"host,optional"`
	Port     int    `hcl:"port,optional"`
	User     string `hcl:"user,optional"`
	Password string `hcl:"password,optional"`
	NoTLS    bool   `hcl:"no_tls,optional"`
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

func loadConfig() (*config, []string) {
	var configFile, host, user, password string
	var port int
	var noTLS bool

	fs := flag.NewFlagSet(fmt.Sprintf("%s %s", os.Args[0], os.Args[1]), flag.ExitOnError)
	fs.StringVar(&configFile, "config", "", "HCL config file")
	fs.StringVar(&host, "host", "", "POP3 server host")
	fs.IntVar(&port, "port", 0, "POP3 server port")
	fs.StringVar(&user, "user", "", "username")
	fs.StringVar(&password, "password", "", "password")
	fs.BoolVar(&noTLS, "no-tls", false, "disable TLS")
	fs.Parse(os.Args[2:])

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
		case "host":
			cfg.Host = host
		case "port":
			cfg.Port = port
		case "user":
			cfg.User = user
		case "password":
			cfg.Password = password
		case "no-tls":
			cfg.NoTLS = noTLS
		}
	})

	if cfg.Port == 0 {
		cfg.Port = 995
	}

	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" {
		fatal(errors.New("host, user, and password are required via config or flag"))
	}

	// XXX: if verbose
	// fmt.Printf("%s:%d %s tls:%v\n", cfg.Host, cfg.Port, cfg.User, !cfg.NoTLS)
	return cfg, fs.Args()
}

func (cfg *config) newConn() *pop3.Conn {
	conn, err := pop3.New(pop3.Opt{
		Host:       cfg.Host,
		Port:       cfg.Port,
		TLSEnabled: !cfg.NoTLS,
	}).NewConn()
	if err != nil {
		fatal(err)
	}

	err = conn.Auth(cfg.User, cfg.Password)
	if err != nil && !strings.Contains(cfg.User, "@") {
		err = conn.Auth(fmt.Sprintf("%s@%s", cfg.User, cfg.Host), cfg.Password)
	}
	if err != nil {
		fatal(err)
	}

	return conn
}
