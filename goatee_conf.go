package main

import (
	"log"
	"os"
	"path"

	"github.com/BurntSushi/toml"
)

var conf struct {
	UI struct {
		MenuBarVisible   bool `toml:"menubar-visible"`
		StatusBarVisible bool `toml:"statusbar-visible"`
	}
	TextView struct {
		Font           string
		LineHightlight bool `toml:"line-hightlight"`
		LineNumbers    bool `toml:"line-numbers"`
		WordWrap       bool `toml:"word-wrap"`
		IndentSpace    bool `toml:"indent-space"`
		IndentWidth    int  `toml:"indent-width"`
	}
	Tabs struct {
		Homogenous bool
		FGNormal   [3]int `toml:"fg-normal"`
		FGModified [3]int `toml:"fg-modified"`
		FGNew      [3]int `toml:"fg-new"`
	}
	Search struct {
		MaxItems int `toml:"max-items"`
	}
	Hex struct {
		BytesInLine int `toml:"bytes-in-line"`
	}
}

func ReadConf() {
	// default values
	conf.UI.MenuBarVisible = false
	conf.UI.StatusBarVisible = false

	conf.TextView.Font = "Liberation Mono 8"
	conf.TextView.LineHightlight = true
	conf.TextView.LineNumbers = true
	conf.TextView.WordWrap = true
	conf.TextView.IndentSpace = false
	conf.TextView.IndentWidth = 2

	conf.Tabs.Homogenous = true
	conf.Tabs.FGNormal = [3]int{200, 200, 200}
	conf.Tabs.FGModified = [3]int{220, 20, 20}
	conf.Tabs.FGNew = [3]int{250, 200, 10}

	conf.Search.MaxItems = 1024

	conf.Hex.BytesInLine = 16

	for _, configfile := range []string{
		path.Join(os.Getenv("XDG_CONFIG_HOME"), "goatee", "goatee.conf"),
		path.Join(os.Getenv("HOME"), ".config", "goatee", "goatee.conf"),
		"goatee.conf",
	} {
		if _, err := os.Stat(configfile); err != nil {
			continue
		}

		_, err := toml.DecodeFile(configfile, &conf)
		if err != nil {
			log.Println("failed decode config file", configfile, "reason:", err)
			continue
		} else {
			return
		}
	}
}
