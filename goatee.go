package main

import (
	"fmt"
	"log"
	"math"
	"net/url"
	"os/user"
	"path"
	"strings"

	arg "github.com/alexflint/go-arg"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
	gsv "github.com/mattn/go-gtk/gtksourceview"
)

var (
	ui   *UI
	conf *Conf

	gvfsPath = "/run/user/%s/gvfs/"

	newtabiter int

	languages = gsv.SourceLanguageManagerGetDefault().GetLanguageIds()
	charsets  = []string{
		CHARSET_UTF8,
		"utf-16",
		"",
		"ISO-8859-2",
		"ISO-8859-7",
		"ISO-8859-9",
		"ISO-8859-15",
		"ShiftJIS",
		"EUC-KR",
		"gb18030",
		"Big5",
		"TIS-620",
		"KOI8-R",
		"",
		"windows-874",
		"windows-1250",
		"windows-1251",
		"windows-1252",
		"windows-1253",
		"windows-1254",
		"windows-1255",
		"windows-1256",
		"windows-1257",
		"windows-1258",
		"",
		CHARSET_BINARY}
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

func xmlEncodings() string {
	var xmldata []string
	for _, c := range charsets {
		if len(c) == 0 {
			xmldata = append(xmldata, "<separator />")
		} else {
			xmldata = append(xmldata, "<menuitem action='"+c+"' />")
		}
	}
	return strings.Join(xmldata, "\n")
}

func errorMessage(err error) {
	m := gtk.NewMessageDialogWithMarkup(nil, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, err.Error())
	m.Run()
	m.Destroy()
}

func convertColor(col [3]int) *gdk.Color {
	r := uint16(math.Pow(float64(col[0]), 2))
	g := uint16(math.Pow(float64(col[1]), 2))
	b := uint16(math.Pow(float64(col[2]), 2))

	return gdk.NewColorRGB(r, g, b)
}

func resolveFilename(filename string) (string, error) {
	if len(filename) == 0 {
		return filename, nil
	}

	u, err := url.Parse(filename)
	if err != nil {
		return filename, fmt.Errorf("failed parse path `%s`, reason: %s", filename, err)
	}

	if len(u.Scheme) == 0 {
		//
	} else if u.Scheme == "file" {
		filename = u.Path
	} else {
		filename = path.Join(gvfsPath, fmt.Sprintf("%s:host=%s", u.Scheme, u.Host), u.Path)
	}

	return filename, nil
}
