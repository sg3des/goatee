package main

import (
	"fmt"
	"log"
	"os/user"

	arg "github.com/alexflint/go-arg"
	"github.com/mattn/go-gtk/gtk"
	gsv "github.com/mattn/go-gtk/gtksourceview"
)

var (
	ui   *UI
	conf *Conf

	gvfsPath = "/run/user/%s/gvfs/"

	newtabiter int

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

	conf = NewConf()

	if len(args.Files) == 0 {
		args.Files = append(args.Files, "")
	}
}

func main() {
	gtk.Init(nil)
	ui = CreateUI()

	for _, filename := range args.Files {
		ui.NewTab(filename)
	}

	gtk.Main()
}

func issetLanguage(lang string) bool {
	for _, langID := range languages {
		if langID == lang {
			return true
		}
	}
	return false
}

func errorMessage(err error) {
	m := gtk.NewMessageDialogWithMarkup(nil, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, err.Error())
	m.Run()
	m.Destroy()
}
