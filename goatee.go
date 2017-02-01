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

	ReadConf()

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

func changeEncoding(data []byte, to, from string) ([]byte, error) {
	cd, err := iconv.Open(to, from) // convert to utf8
	if err != nil {
		return nil, fmt.Errorf("unknown charsets: `%s` `%s`, %s", to, from, err)
	}
	defer cd.Close()

	var outbuf = make([]byte, len(data))
	out, _, err := cd.Conv(data, outbuf)
	if err != nil {
		return nil, fmt.Errorf("failed change encoding from `%s`, %s", from, err)
	}
	return out, nil
}

func errorMessage(err error) {
	m := gtk.NewMessageDialogWithMarkup(nil, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, err.Error())
	m.Run()
	m.Destroy()
}
