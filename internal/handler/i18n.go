package handler

import (
	"io/fs"
	"path/filepath"
	"regexp"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

var defaultBundle *i18n.Bundle
var activeRegex = regexp.MustCompile(`active\.[\w]+(?:\-\w+)?\.yaml`)

func init() {
	defaultBundle = i18n.NewBundle(language.English)
	defaultBundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	err := filepath.WalkDir("locales", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && activeRegex.MatchString(d.Name()) {
			_, err := defaultBundle.LoadMessageFile(path)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}
}

func GetLocalizer(lang ...string) *i18n.Localizer {
	return i18n.NewLocalizer(defaultBundle, lang...)
}
