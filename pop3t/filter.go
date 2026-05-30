package main

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"

	msgformat "github.com/emersion/go-message"
	"github.com/knadh/go-pop3"
	"github.com/pemistahl/lingua-go"
)

var (
	languages = map[string]lingua.Language{
		"afrikaans":   lingua.Afrikaans,
		"albanian":    lingua.Albanian,
		"arabic":      lingua.Arabic,
		"armenian":    lingua.Armenian,
		"azerbaijani": lingua.Azerbaijani,
		"basque":      lingua.Basque,
		"belarusian":  lingua.Belarusian,
		"bengali":     lingua.Bengali,
		"bokmal":      lingua.Bokmal,
		"bosnian":     lingua.Bosnian,
		"bulgarian":   lingua.Bulgarian,
		"catalan":     lingua.Catalan,
		"chinese":     lingua.Chinese,
		"croatian":    lingua.Croatian,
		"czech":       lingua.Czech,
		"danish":      lingua.Danish,
		"dutch":       lingua.Dutch,
		"english":     lingua.English,
		"esperanto":   lingua.Esperanto,
		"estonian":    lingua.Estonian,
		"finnish":     lingua.Finnish,
		"french":      lingua.French,
		"ganda":       lingua.Ganda,
		"georgian":    lingua.Georgian,
		"german":      lingua.German,
		"greek":       lingua.Greek,
		"gujarati":    lingua.Gujarati,
		"hebrew":      lingua.Hebrew,
		"hindi":       lingua.Hindi,
		"hungarian":   lingua.Hungarian,
		"icelandic":   lingua.Icelandic,
		"indonesian":  lingua.Indonesian,
		"irish":       lingua.Irish,
		"italian":     lingua.Italian,
		"japanese":    lingua.Japanese,
		"kazakh":      lingua.Kazakh,
		"korean":      lingua.Korean,
		"latin":       lingua.Latin,
		"latvian":     lingua.Latvian,
		"lithuanian":  lingua.Lithuanian,
		"macedonian":  lingua.Macedonian,
		"malay":       lingua.Malay,
		"maori":       lingua.Maori,
		"marathi":     lingua.Marathi,
		"mongolian":   lingua.Mongolian,
		"nynorsk":     lingua.Nynorsk,
		"persian":     lingua.Persian,
		"polish":      lingua.Polish,
		"portuguese":  lingua.Portuguese,
		"punjabi":     lingua.Punjabi,
		"romanian":    lingua.Romanian,
		"russian":     lingua.Russian,
		"serbian":     lingua.Serbian,
		"shona":       lingua.Shona,
		"slovak":      lingua.Slovak,
		"slovene":     lingua.Slovene,
		"somali":      lingua.Somali,
		"sotho":       lingua.Sotho,
		"spanish":     lingua.Spanish,
		"swahili":     lingua.Swahili,
		"swedish":     lingua.Swedish,
		"tagalog":     lingua.Tagalog,
		"tamil":       lingua.Tamil,
		"telugu":      lingua.Telugu,
		"thai":        lingua.Thai,
		"tsonga":      lingua.Tsonga,
		"tswana":      lingua.Tswana,
		"turkish":     lingua.Turkish,
		"ukrainian":   lingua.Ukrainian,
		"urdu":        lingua.Urdu,
		"vietnamese":  lingua.Vietnamese,
		"welsh":       lingua.Welsh,
		"xhosa":       lingua.Xhosa,
		"yoruba":      lingua.Yoruba,
		"zulu":        lingua.Zulu,
	}
)

func parseLanguageFilter(s string) (lingua.Language, bool, error) {
	exclude := strings.HasPrefix(s, "^")
	if exclude {
		s = strings.TrimLeft(s, "^")
	}

	lang, ok := languages[strings.ToLower(s)]
	if !ok {
		return lingua.Unknown, false, fmt.Errorf("unknown language: %s", s)
	}
	return lang, exclude, nil
}

func matchLanguage(lang lingua.Language, langFilters []lingua.Language, exclude bool) bool {
	for _, langFilter := range langFilters {
		if lang == langFilter {
			return !exclude
		}
	}

	return len(langFilters) == 0 || exclude
}

func filter(cfg *config, args []string) {
	var del bool
	var addrs []string
	var langFilters []lingua.Language
	var exclude bool

	for _, arg := range args {
		switch {
		case arg == "delete":
			if del {
				fatal(errors.New("delete specified more than once"))
			}
			del = true
		case strings.HasPrefix(arg, "forward="):
			addrs = append(addrs, strings.TrimPrefix(arg, "forward="))
		default:
			lang, langExclude, err := parseLanguageFilter(arg)
			if err != nil {
				fatal(err)
			} else if len(langFilters) > 0 && langExclude != exclude {
				fatal(fmt.Errorf("language filters must be all positive or all negative: %s",
					arg))
			}
			exclude = langExclude
			langFilters = append(langFilters, lang)
		}
	}

	var smtpAddr string
	var smtpAuth smtp.Auth
	if len(addrs) > 0 {
		var err error
		smtpAddr, smtpAuth, err = cfg.newSend()
		if err != nil {
			fatal(err)
		}
	}

	cnt := 0
	ld := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	tot, err := cfg.list(func(conn *pop3.Conn, id int, entity *msgformat.Entity) error {
		msg, err := messageFromEntity(entity)
		if err != nil {
			fmt.Printf("skipping: entity(%d): %s\n", id, err)
			return nil
		}
		lang, conf, exists := msg.detectLanguage(ld)
		if !exists || conf < 0.5 || !matchLanguage(lang, langFilters, exclude) {
			return nil
		}
		fmt.Printf("%3d [%s %.0f%%] %s\n", id, lang, conf*100, msg.subject)

		if len(addrs) > 0 {
			buf, err := conn.Cmd("RETR", true, id)
			if err != nil {
				return err
			}
			err = smtp.SendMail(smtpAddr, smtpAuth, cfg.smtpUser(), addrs, buf.Bytes())
			if err != nil {
				return err
			}
			fmt.Printf("forwarded %d to %s\n", id, strings.Join(addrs, ", "))
		}
		if del {
			path, err := deleteId(cfg, conn, id)
			if err != nil {
				return err
			}
			fmt.Printf("deleted %d", id)
			if path != "" {
				fmt.Printf(" (saved to %s)", path)
			}
			fmt.Println()
		}

		cnt += 1
		return nil
	})
	if err != nil {
		fatal(err)
	}

	fmt.Printf("%d of %d messages\n", cnt, tot)
}
