package main

import (
	"log"
	"os"
	"path"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
)

//conf structure contains configuration
type Conf struct {
	window *gtk.Window

	UI struct {
		MenuBarVisible   bool `toml:"menubar-visible"`
		StatusBarVisible bool `toml:"statusbar-visible"`
	}
	TextView struct {
		Font           string
		LineHightlight bool   `toml:"line-hightlight"`
		LineNumbers    bool   `toml:"line-numbers"`
		WordWrap       bool   `toml:"word-wrap"`
		IndentSpace    bool   `toml:"indent-space"`
		IndentWidth    int    `toml:"indent-width"`
		StyleScheme    string `toml:"style-scheme"`
	}
	Tabs struct {
		Homogenous bool
		CloseBtns  bool `toml:"close-buttons"`
		Height     int
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

//NewConf set default values for configuration and parse config file
func NewConf() *Conf {
	// default values
	conf := new(Conf)
	conf.UI.MenuBarVisible = false
	conf.UI.StatusBarVisible = false

	conf.TextView.Font = "Liberation Mono 8"
	conf.TextView.LineHightlight = true
	conf.TextView.LineNumbers = true
	conf.TextView.WordWrap = true
	conf.TextView.IndentSpace = false
	conf.TextView.IndentWidth = 2

	conf.Tabs.Homogenous = true
	conf.Tabs.CloseBtns = true
	conf.Tabs.Height = 16
	conf.Tabs.FGNormal = [3]int{200, 200, 200}
	conf.Tabs.FGModified = [3]int{220, 20, 20}
	conf.Tabs.FGNew = [3]int{250, 200, 10}

	conf.Search.MaxItems = 1024

	conf.Hex.BytesInLine = 16

	//parse config files
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
		}
		break
	}

	conf.CreateWindow()

	return conf
}

//OpenWindow open window configuration
func (c *Conf) OpenWindow() {
	if c.window == nil {
		c.CreateWindow()
	}
	c.window.ShowAll()
}

//CreateWindow create window configuration
func (c *Conf) CreateWindow() {
	c.window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	c.window.SetTypeHint(gdk.WINDOW_TYPE_HINT_DIALOG)
	c.window.SetDefaultSize(300, 300)
	c.window.SetSizeRequest(300, 300)

	rc := reflect.TypeOf(*c)
	// rc := reflect.ValueOf(*c)
	for i := 0; i < rc.NumField(); i++ {
		log.Println(rc.Field(i).Type)
	}

	vbox := gtk.NewVBox(false, 0)

	c.window.Add(vbox)
}
