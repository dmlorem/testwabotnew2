package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

type ConfigScheme struct {
	BotName       string   `mapstructure:"botname"`
	OwnerNumbers  []string `mapstructure:"owners"`
	CommandPrefix string   `mapstructure:"cmdprefix"`
	CommandsDelay int64    `mapstructure:"commandsdelay"`
	ReadMessages  bool     `mapstructure:"readmessages"`
	StickerTitle  string   `mapstructure:"stickertitle"`
	StickerAuthor string   `mapstructure:"stickerauthor"`
	Language      string   `mapstructure:"language"`
	PairWithCode  bool     `mapstructure:"pairwithcode"`

	v *viper.Viper
}

func LoadConfig(configPath string) (*ConfigScheme, error) {
	c := &ConfigScheme{}
	v := &viper.Viper{}
	v = viper.New()

	v.SetConfigType(strings.ReplaceAll(filepath.Ext(configPath), ".", ""))
	v.SetConfigName(strings.ReplaceAll(filepath.Base(configPath), filepath.Ext(configPath), ""))
	v.AddConfigPath(filepath.Dir(configPath))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Error reading config: %w", err)
	}
	if err := v.Unmarshal(c); err != nil {
		return nil, fmt.Errorf("Error unmarshalling config: %w", err)
	}

	v.WatchConfig()
	return c, nil
}

func (c *ConfigScheme) SaveConfig() error {
	val := reflect.ValueOf(c).Elem()
	typ := val.Type()

	for i := range val.NumField() {
		field := typ.Field(i)
		key := field.Tag.Get("mapstructure")

		if key == "" {
			continue
		}

		c.v.Set(key, val.Field(i).Interface())
	}

	return c.v.WriteConfig()
}
