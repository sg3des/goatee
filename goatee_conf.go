package main

import (
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
	"github.com/naoina/toml"
)

//Conf structure contains configuration
type Conf struct {
	window   *gtk.Window `toml:",omitempty"`
	filename string      `toml:",omitempty"`

	UI struct {
		MenuBarVisible   bool `toml:"menubar-visible"`
		StatusBarVisible bool `toml:"statusbar-visible"`
	}
	TextView struct {
		Font           string `toml:"font"`
		LineHightlight bool   `toml:"line-hightlight"`
		LineNumbers    bool   `toml:"line-numbers"`
		WordWrap       bool   `toml:"word-wrap"`
		IndentSpace    bool   `toml:"indent-space"`
		IndentWidth    int    `toml:"indent-width"`
		StyleScheme    string `toml:"style-scheme"`
	}
	Tabs struct {
		Homogeneous bool  `toml:"homogeneous"`
		CloseBtns   bool  `toml:"close-buttons"`
		Height      int   `toml:"height"`
		FGNormal    []int `toml:"fg-normal"`
		FGModified  []int `toml:"fg-modified"`
		FGNew       []int `toml:"fg-new"`
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
	c := new(Conf)
	c.UI.MenuBarVisible = true
	c.UI.StatusBarVisible = false

	c.TextView.Font = "Liberation Mono 8"
	c.TextView.LineHightlight = true
	c.TextView.LineNumbers = true
	c.TextView.WordWrap = true
	c.TextView.IndentSpace = false
	c.TextView.IndentWidth = 2

	c.Tabs.Homogeneous = true
	c.Tabs.CloseBtns = true
	c.Tabs.Height = 24
	c.Tabs.FGNormal = []int{200, 200, 200}
	c.Tabs.FGModified = []int{220, 20, 20}
	c.Tabs.FGNew = []int{250, 200, 10}

	c.Search.MaxItems = 1024

	c.Hex.BytesInLine = 16

	//parse config files
	for _, configfile := range []string{
		path.Join(os.Getenv("XDG_CONFIG_HOME"), "goatee", "goatee.conf"),
		path.Join(os.Getenv("HOME"), ".config", "goatee", "goatee.conf"),
		"goatee.conf",
	} {
		data, err := ioutil.ReadFile(configfile)
		if err != nil {
			continue
		}

		err = toml.Unmarshal(data, &c)
		if err != nil {
			log.Printf("failed decode config file '%s', reason: %s", configfile, err)
			continue
		}

		c.filename = configfile
		break
	}

	return c
}

func (c *Conf) Write() {
	c.CreateDirConfig()

	log.Println("write", c.filename)

	if c.filename == "" {
		return
	}

	data, err := toml.Marshal(&c)
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(c.filename, data, 0644)
	if err != nil {
		log.Println(err)
		return
	}
}

func (c *Conf) CreateDirConfig() {
	//if file already created nothing do
	if len(c.filename) > 0 {
		return
	}

	dirConf := os.Getenv("XDG_CONFIG_HOME")
	dirHome := os.Getenv("HOME")

	if len(dirConf) == 0 && len(dirHome) != 0 {
		dirConf = path.Join(dirHome, ".config")
		stat, err := os.Stat(dirConf)
		if err != nil || stat.IsDir() {
			return
		}
	}

	if len(dirConf) == 0 {
		return
	}

	dirGoateeConf := path.Join(dirConf, "goatee")
	err := os.MkdirAll(dirGoateeConf, 0700)
	if err != nil {
		log.Println("failed create directory for save configuration,reason:", err)
		return
	}
	c.filename = path.Join(dirGoateeConf, "goatee.conf")
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
	c.window.SetName("Preferences")
	c.window.SetTypeHint(gdk.WINDOW_TYPE_HINT_DIALOG)
	c.window.SetDefaultSize(300, 300)
	c.window.SetSizeRequest(300, 300)

	vbox := gtk.NewVBox(false, 0)

	notebook := gtk.NewNotebook()

	rc := reflect.TypeOf(*conf)
	rv := reflect.ValueOf(conf).Elem()
	for i := 0; i < rc.NumField(); i++ {
		if rc.Field(i).Type.Kind() != reflect.Struct {
			continue
		}

		fvbox := gtk.NewVBox(false, 0)
		notebook.AppendPage(fvbox, gtk.NewLabel(rc.Field(i).Name))

		confStruct := rc.Field(i).Type
		for j := 0; j < confStruct.NumField(); j++ {
			field := confStruct.Field(j)
			val := rv.Field(i).Field(j)

			//prepare name field
			name := strings.Split(field.Tag.Get("toml"), ",")[0]
			if len(name) == 0 {
				name = field.Name
			}
			name = c.FormatName(name)

			hbox := gtk.NewHBox(false, 0)

			label := gtk.NewLabel(name)
			hbox.PackStart(label, false, false, 5)

			switch val.Kind() {
			case reflect.Bool:
				widget := gtk.NewCheckButton()
				widget.SetName(field.Name)
				widget.SetSizeRequest(120, 20)
				widget.SetActive(val.Bool())

				w := &ConfWidget{chkbtn: widget, Field: val, conf: c}
				widget.Connect("clicked", w.UpdateValue)

				hbox.PackEnd(widget, false, false, 5)
			case reflect.String:
				widget := gtk.NewEntry()
				widget.SetName(field.Name)
				widget.SetSizeRequest(120, 20)
				widget.SetText(val.String())

				w := &ConfWidget{entry: widget, Field: val, conf: c}
				widget.Connect("changed", w.UpdateValue)

				hbox.PackEnd(widget, false, false, 5)
			case reflect.Int:
				widget := gtk.NewSpinButtonWithRange(-1, 2048, 1)
				widget.SetName(field.Name)
				widget.SetSizeRequest(120, 20)
				widget.SetValue(float64(val.Int()))

				w := &ConfWidget{spnbtn: widget, Field: val, conf: c}
				widget.Connect("changed", w.UpdateValue)

				hbox.PackEnd(widget, false, false, 5)
			case reflect.TypeOf([]int{}).Kind(): //color
				color := convertColor(val.Interface().([]int))
				widget := gtk.NewColorButtonWithColor(color)
				widget.SetName(field.Name)
				widget.SetSizeRequest(120, 20)

				w := &ConfWidget{colbtn: widget, Field: val, conf: c}
				widget.Connect("color-set", w.UpdateValue)

				hbox.PackEnd(widget, false, false, 5)
			}

			fvbox.PackStart(hbox, false, false, 5)
		}
	}

	closebtn := gtk.NewButtonFromStock(gtk.STOCK_CLOSE)
	closebtn.Clicked(c.CloseWindow)
	hbox := gtk.NewHBox(false, 0)
	hbox.PackEnd(closebtn, false, false, 5)

	vbox.Add(notebook)
	vbox.PackEnd(hbox, false, false, 5)

	c.window.Add(vbox)
}

func (c *Conf) CloseWindow() {
	c.Write()

	c.window.Hide()
}

type ConfWidget struct {
	Field reflect.Value

	conf *Conf

	chkbtn *gtk.CheckButton
	entry  *gtk.Entry
	spnbtn *gtk.SpinButton
	colbtn *gtk.ColorButton
}

func (w *ConfWidget) UpdateValue() {
	switch {
	case w.chkbtn != nil:
		w.Field.SetBool(w.chkbtn.GetActive())
	case w.entry != nil:
		w.Field.SetString(w.entry.GetText())
	case w.spnbtn != nil:
		n, _ := strconv.Atoi(w.spnbtn.Entry.GetText())
		w.Field.SetInt(int64(n))
	case w.colbtn != nil:
		col := w.colbtn.GetColor()
		r := int(math.Sqrt(float64(col.Red())))
		g := int(math.Sqrt(float64(col.Green())))
		b := int(math.Sqrt(float64(col.Blue())))

		w.Field.Set(reflect.ValueOf([3]int{r, g, b}))
	}
}

func (c *Conf) FormatName(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = regexp.MustCompile("([A-Z])").ReplaceAllString(name, " $1")
	name = strings.TrimSpace(name)
	name = strings.Title(name)
	return name
}
