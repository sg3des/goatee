package main

import (
	"fmt"
	"log"
	"os/user"

	arg "github.com/alexflint/go-arg"
	"github.com/mattn/go-gtk/gtk"
	gsv "github.com/mattn/go-gtk/gtksourceview"

	iconv "gopkg.in/iconv.v1"
)

var (
	ui *UI

	gvfsPath = "/run/user/%s/gvfs/"

	width  int
	height int

	newtabiter int
	tabs       []*Tab

	languages = gsv.SourceLanguageManagerGetDefault().GetLanguageIds()
)

var args struct {
	Files []string `arg:"positional"`
}

func init() {
	log.SetFlags(log.Lshortfile)
	arg.MustParse(&args)

	user, _ := user.Current()
	gvfsPath = fmt.Sprintf(gvfsPath, user.Uid)
	ReadConf()
}

func main() {
	gtk.Init(nil)
	ui = CreateUI()

	if len(args.Files) > 0 {
		for _, filename := range args.Files {
			NewTab(filename)
		}
	} else {
		NewTab("")
	}

	gtk.Main()
}

func tabsContains(filename string) bool {
	for n, t := range tabs {
		if t.Filename == filename {
			ui.notebook.SetCurrentPage(n)
			return true
		}
	}
	return false
}

func issetLanguage(lang string) bool {
	for _, langID := range languages {
		if langID == lang {
			return true
		}
	}
	return false
}

func changeEncoding(data []byte, to, from string) ([]byte, error) {
	cd, err := iconv.Open(to, from) // convert to utf8
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
	n := ui.notebook.GetCurrentPage()
	ui.notebook.RemovePage(tabs[n].swin, n)
	tabs[n].File.Close()
	tabs = append(tabs[:n], tabs[n+1:]...)
}

func currentTab() *Tab {
	n := ui.notebook.GetCurrentPage()
	if n < 0 {
		return nil
	}
	return tabs[n]
}

func errorMessage(err error) {
	m := gtk.NewMessageDialogWithMarkup(nil, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, err.Error())
	m.Run()
	m.Destroy()

}
