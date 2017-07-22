package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
	gsv "github.com/mattn/go-gtk/gtksourceview"

	"github.com/naoina/toml"
)

//Conf structure contains configuration
type Conf struct {
	window   *gtk.Window `toml:",omitempty"`
	filename string      `toml:",omitempty"`

	UI struct {
		MenuBarVisible   bool `toml:"menubar-visible" wgt:"checkbox"`
		StatusBarVisible bool `toml:"statusbar-visible" wgt:"checkbox"`
	}
	TextView struct {
		Font           string `toml:"font" wgt:"font"`
		LineHightlight bool   `toml:"line-hightlight" wgt:"checkbox"`
		LineNumbers    bool   `toml:"line-numbers" wgt:"checkbox"`
		WordWrap       bool   `toml:"word-wrap" wgt:"checkbox"`
		IndentSpace    bool   `toml:"indent-space" wgt:"checkbox"`
		IndentWidth    int    `toml:"indent-width" wgt:"int"`
		StyleScheme    string `toml:"style-scheme" wgt:"styles"`
	}
	Tabs struct {
		Homogeneous bool  `toml:"homogeneous" wgt:"checkbox"`
		CloseBtns   bool  `toml:"close-buttons" wgt:"checkbox"`
		Height      int   `toml:"height" wgt:"int"`
		FGNormal    []int `toml:"fg-normal" wgt:"color"`
		FGModified  []int `toml:"fg-modified" wgt:"color"`
		FGNew       []int `toml:"fg-new" wgt:"color"`
	}
	Search struct {
		MaxItems int `toml:"max-items" wgt:"int"`
	}
	Hex struct {
		BytesInLine int `toml:"bytes-in-line" wgt:"int"`
	}
}

//NewConf set default values for configuration and parse config file
func NewConf() *Conf {
	confdir := os.Getenv("XDG_CONFIG_HOME")
	if confdir == "" {
		confdir = path.Join(os.Getenv("HOME"), ".config")
	}
	confdir = path.Join(confdir, "goatee")

	configfiles := []string{
		path.Join(confdir, "goatee.conf"),
		"goatee.conf",
	}

	// default values
	c := new(Conf)
	c.filename = path.Join(confdir, "goatee.conf")
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
	c.Tabs.Height = 16
	c.Tabs.FGNormal = []int{200, 200, 200}
	c.Tabs.FGModified = []int{220, 20, 20}
	c.Tabs.FGNew = []int{250, 200, 10}

	c.Search.MaxItems = 1024

	c.Hex.BytesInLine = 16

	//parse config files
	for _, filename := range configfiles {
		if err := c.readConfigFile(filename); err == nil {
			c.filename = filename
			break
		}
	}

	return c
}

func (c *Conf) readConfigFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if err = toml.Unmarshal(data, &c); err != nil {
		err = fmt.Errorf("failed decode config file '%s', reason: %s", filename, err)
		log.Println(err)
		return err
	}

	return nil
}

func (c *Conf) Write() {
	os.MkdirAll(filepath.Dir(c.filename), 0755)

	log.Println("write config to:", c.filename)

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

	c.readConfigFile(c.filename)

	t := reflect.TypeOf(c).Elem()
	v := reflect.ValueOf(c).Elem()
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type.Kind() != reflect.Struct {
			continue
		}

		fvbox := gtk.NewVBox(false, 0)
		notebook.AppendPage(fvbox, gtk.NewLabel(t.Field(i).Name))

		confStruct := t.Field(i).Type
		for j := 0; j < confStruct.NumField(); j++ {
			field := confStruct.Field(j)
			val := v.Field(i).Field(j)

			label, widget := c.newWidget(val, field)

			hbox := gtk.NewHBox(false, 0)
			hbox.PackStart(label, false, false, 5)
			hbox.PackEnd(widget, false, false, 5)

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

func (c *Conf) newWidget(v reflect.Value, f reflect.StructField) (*gtk.Label, gtk.IWidget) {
	name := c.getFieldName(f)
	label := gtk.NewLabel(name)

	tag, ok := f.Tag.Lookup("wgt")
	if !ok {
		log.Fatalf("tag `wgt` not set for field %s", name)
	}

	w := &ConfWidget{Field: v, conf: c}

	switch tag {
	case "checkbox":
		w.chkbtn = gtk.NewCheckButton()
		w.chkbtn.SetSizeRequest(150, -1)
		w.chkbtn.SetActive(v.Bool())
		w.chkbtn.Connect("clicked", w.UpdateValue)

	case "string":
		w.entry = gtk.NewEntry()
		w.entry.SetSizeRequest(150, -1)
		w.entry.SetText(v.String())
		w.entry.Connect("changed", w.UpdateValue)

	case "int":
		w.spnbtn = gtk.NewSpinButtonWithRange(-1, 2048, 1)
		w.spnbtn.SetSizeRequest(150, -1)
		w.spnbtn.SetValue(float64(v.Int()))
		w.spnbtn.Connect("changed", w.UpdateValue)

	case "color":
		color := convertColor(v.Interface().([]int))
		w.colbtn = gtk.NewColorButtonWithColor(color)
		w.colbtn.SetSizeRequest(150, -1)
		w.colbtn.Connect("color-set", w.UpdateValue)

	case "font":
		w.fntbtn = gtk.NewFontButton()
		w.fntbtn.SetSizeRequest(150, -1)
		w.fntbtn.SetFontName(v.String())
		w.fntbtn.Connect("font-set", w.UpdateValue)

	case "styles":
		schemes := gsv.SourceStyleSchemeManagerGetDefault().GetSchemeIds()
		scheme := v.String()

		w.cmbbox = gtk.NewComboBoxText()
		w.cmbbox.SetSizeRequest(150, -1)
		for i, s := range schemes {
			w.cmbbox.AppendText(s)
			if scheme == s {
				w.cmbbox.SetActive(i)
			}
		}
		w.cmbbox.Connect("changed", w.UpdateValue)
	}

	return label, w.GetWidget()
}

func (c *Conf) getFieldName(f reflect.StructField) string {
	name := strings.Split(f.Tag.Get("toml"), ",")[0]
	if len(name) == 0 {
		name = f.Name
	}
	return c.FormatName(name)
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
	fntbtn *gtk.FontButton
	cmbbox *gtk.ComboBoxText
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

		w.Field.Set(reflect.ValueOf([]int{r, g, b}))
	case w.fntbtn != nil:
		w.Field.SetString(w.fntbtn.GetFontName())
	case w.cmbbox != nil:
		w.Field.SetString(w.cmbbox.GetActiveText())
	}
}

func (w *ConfWidget) GetWidget() gtk.IWidget {
	switch {
	case w.chkbtn != nil:
		return w.chkbtn
	case w.entry != nil:
		return w.entry
	case w.spnbtn != nil:
		return w.spnbtn
	case w.colbtn != nil:
		return w.colbtn
	case w.fntbtn != nil:
		return w.fntbtn
	case w.cmbbox != nil:
		return w.cmbbox
	}
	log.Println(w)
	return nil
}

func (c *Conf) FormatName(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = regexp.MustCompile("([A-Z])").ReplaceAllString(name, " $1")
	name = strings.TrimSpace(name)
	name = strings.Title(name)
	return name
}
