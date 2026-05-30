package main

import (
	"errors"
	"flag"
	"fmt"
	"net/smtp"
	"os"
	"runtime"
	"strconv"
	"strings"

	msgformat "github.com/emersion/go-message"
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
	User     string     `hcl:"user,optional"`
	Password string     `hcl:"password,optional"`
	Archive  string     `hcl:"archive,optional"`
	POP3     pop3Config `hcl:"pop3,optional"`
	SMTP     smtpConfig `hcl:"smtp,optional"`
}

func (cfg *config) pop3Host() string {
	if cfg.POP3.Host != "" {
		return cfg.POP3.Host
	}
	return cfg.Host
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

func (cfg *config) smtpHost() string {
	if cfg.SMTP.Host != "" {
		return cfg.SMTP.Host
	}
	return cfg.Host
}

func (cfg *config) smtpUser() string {
	if cfg.SMTP.User != "" {
		return cfg.SMTP.User
	}
	return cfg.User
}

func (cfg *config) smtpPassword() string {
	if cfg.SMTP.Password != "" {
		return cfg.SMTP.Password
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

	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")
	fs.BoolVar(&verbose, "v", false, "enable verbose output")
	fs.StringVar(&configFile, "config", "", "HCL config file")
	fs.String("archive", "", "directory to save deleted emails")
	fs.String("pop3-host", "", "POP3 server host")
	fs.Int("pop3-port", 0, "POP3 server port")
	fs.String("pop3-user", "", "POP3 username")
	fs.String("pop3-password", "", "POP3 password")
	fs.Bool("pop3-no-tls", false, "POP3 disable TLS")
	fs.String("smtp-host", "", "SMTP server host")
	fs.Int("smtp-port", 0, "SMTP server port")
	fs.String("smtp-user", "", "SMTP username")
	fs.String("smtp-password", "", "SMTP password")
	fs.Bool("smtp-no-tls", false, "SMTP disable TLS")
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
		case "smtp-host":
			cfg.SMTP.Host = f.Value.String()
		case "smtp-port":
			cfg.SMTP.Port, _ = strconv.Atoi(f.Value.String())
		case "smtp-user":
			cfg.SMTP.User = f.Value.String()
		case "smtp-password":
			cfg.SMTP.Password = f.Value.String()
		case "smtp-no-tls":
			cfg.SMTP.NoTLS = (f.Value.String() == "true")
		case "archive":
			cfg.Archive = f.Value.String()
		}
	})

	return cfg, fs.Args()
}

func (cfg *config) list(fn func(conn *pop3.Conn, id int, entity *msgformat.Entity) error) (int,
	error) {

	conn, err := cfg.newConn()
	if err != nil {
		return 0, err
	}
	defer conn.Quit()

	mids, err := conn.List(0)
	if err != nil {
		conn.Rset()
		return 0, err
	}

	for _, mid := range mids {
		entity, err := conn.Retr(mid.ID)
		if err != nil {
			fmt.Printf("skipping: retr(%d): %s\n", mid.ID, err)
			continue // XX
		}
		err = fn(conn, mid.ID, entity)
		if err != nil {
			conn.Rset()
			return 0, err
		}
	}

	return len(mids), nil
}

func (cfg *config) newConn() (*pop3.Conn, error) {
	host := cfg.pop3Host()
	user := cfg.pop3User()
	password := cfg.pop3Password()
	if host == "" || user == "" || password == "" {
		return nil, errors.New("pop3 host, user, and password are required via config or flag")
	}

	port := cfg.POP3.Port
	if port == 0 {
		if cfg.POP3.NoTLS {
			port = 110
		} else {
			port = 995
		}
	}

	if verbose {
		fmt.Printf("pop3: %s:%d user:%s tls:%v\n", host, port, user, !cfg.POP3.NoTLS)
	}

	conn, err := pop3.New(pop3.Opt{
		Host:       host,
		Port:       port,
		TLSEnabled: !cfg.POP3.NoTLS,
	}).NewConn()
	if err != nil {
		return nil, err
	}

	err = conn.Auth(user, password)
	if err != nil && !strings.Contains(user, "@") {
		err = conn.Auth(fmt.Sprintf("%s@%s", user, host), password)
	}
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (cfg *config) newSend() (string, smtp.Auth, error) {
	host := cfg.smtpHost()
	user := cfg.smtpUser()
	password := cfg.smtpPassword()
	if host == "" || user == "" || password == "" {
		return "", nil, errors.New("smtp host, user, and password are required via config or flag")
	}

	port := cfg.SMTP.Port
	if port == 0 {
		if cfg.SMTP.NoTLS {
			port = 25
		} else {
			port = 587
		}
	}

	smtpAddr := fmt.Sprintf("%s:%d", host, port)
	smtpAuth := smtp.PlainAuth("", user, password, host)
	if verbose {
		fmt.Printf("smtp: %s user:%s tls:%v\n", smtpAddr, user, !cfg.SMTP.NoTLS)
	}

	return smtpAddr, smtpAuth, nil
}
