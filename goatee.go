package main

import (
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/mattn/go-gtk/gtk"
	gsv "github.com/mattn/go-gtk/gtksourceview"

	iconv "gopkg.in/iconv.v1"
)

var (
	gvfsPath = "/run/user/%s/gvfs/"

	width  int
	height int

	window   *gtk.Window
	notebook *gtk.Notebook
	filename string

	newtabiter int
	tabs       []*Tab

	languages = gsv.SourceLanguageManagerGetDefault().GetLanguageIds()

	err error
)

func init() {
	log.SetFlags(log.Lshortfile)

	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	ReadConf()

	user, _ := user.Current()

	gvfsPath = fmt.Sprintf(gvfsPath, user.Uid)
}

func main() {
	gtk.Init(nil)
	window = CreateWindow()
	window.Connect("destroy", exit)
	window.Connect("check-resize", windowResize)

	window.ShowAll()
	menubar.SetVisible(false)
	footer.SetVisible(false)

	NewTab(filename)

	gtk.Main()
}

func exit() {
	for _, t := range tabs {
		t.File.Close()
	}

	gtk.MainQuit()
}

func CreateWindow() *gtk.Window {
	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetDefaultSize(700, 300)
	vbox := gtk.NewVBox(false, 0)
	CreateMenu(window, vbox)
	notebook = gtk.NewNotebook()
	vbox.Add(notebook)
	window.Add(vbox)

	return window
}

func windowResize() {
	window.GetSize(&width, &height)
	notebook.SetSizeRequest(width, height)
	homogenousTabs()
}

func tabsContains(filename string) bool {
	for n, t := range tabs {
		if t.Filename == filename {
			notebook.SetCurrentPage(n)
			return true
		}
	}
	return false
}

func issetLanguage(lang string) bool {
	for _, langId := range languages {
		if langId == lang {
			return true
		}
	}
	return false
}

func changeEncoding(data []byte, to, from string) ([]byte, error) {
	cd, err := iconv.Open(to, from) // convert gbk to utf8
	if err != nil {
		return nil, fmt.Errorf("unknown charsets: `%s` `%s`, %s", to, from, err)
	}
	defer cd.Close()

	var outbuf = make([]byte, len(data))
	out, _, err := cd.Conv(data, outbuf)
	if err != nil {
		return nil, fmt.Errorf("failed convert encoding, %s", err)
	}
	return out, nil
}

func closeCurrentTab() {
	n := notebook.GetCurrentPage()
	notebook.RemovePage(tabs[n].swin, n)
	tabs[n].File.Close()
	tabs = append(tabs[:n], tabs[n+1:]...)
}

func currentTab() *Tab {
	n := notebook.GetCurrentPage()
	if n < 0 {
		return nil
	}
	return tabs[n]
}

func homogenousTabs() {
	if len(tabs) == 0 || !conf.Tabs.Homogenous {
		return
	}

	tabwidth := (width - len(tabs)*6) / len(tabs)
	leftwidth := (width - len(tabs)*6) % len(tabs)

	for _, t := range tabs {
		if leftwidth > 0 {
			t.label.SetSizeRequest(tabwidth+1, 12)
			leftwidth--
		} else {
			t.label.SetSizeRequest(tabwidth, 12)
		}
	}

}
