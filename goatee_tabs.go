package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
	"unsafe"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"

	gsv "github.com/mattn/go-gtk/gtksourceview"
)

type Tab struct {
	Filename string
	File     *os.File
	Data     []byte
	Encoding string
	Language string
	ReadOnly bool

	//color        *gdk.Color
	swin         *gtk.ScrolledWindow
	label        *gtk.Label
	sourcebuffer *gsv.SourceBuffer
	sourceview   *gsv.SourceView

	tagfind *gtk.TextTag
}

func NewTab(filename string) {
	var newfile bool

	// log.Println(filename)
	if tabsContains(filename) {
		return
	}

	if filename == "" {
		filename = fmt.Sprintf("new%d", newtabiter)
		newtabiter++
		newfile = true
	}

	t := &Tab{}
	t.Filename = filename

	if !newfile {
		// t.File, err = os.Open(filename)
		info, err := os.Lstat(filename)
		if err != nil {
			log.Println("failed get stat of file", filename, err)
			return
		}

		t.File, err = os.OpenFile(filename, os.O_RDWR, info.Mode())
		if err != nil {
			t.ReadOnly = true
			t.File, err = os.OpenFile(filename, os.O_RDONLY, info.Mode())

			if err != nil {
				log.Println("failed open file", filename, err)
				return
			}
		}

		t.Data, err = ioutil.ReadAll(t.File)
		if err != nil {
			log.Println("failed read file", filename, err)
		}

		if len(t.Data) > 0 {
			t.Encoding = t.DetectEncoding()

			if t.Encoding != "utf-8" && t.Encoding != "binary" {
				t.Data, err = changeEncoding(t.Data, "utf-8", t.Encoding)
				if err != nil {
					fmt.Println(err)
					return
				}
			}

			if t.Encoding != "binary" {
				t.Language = t.DetectLanguage()
			} else {
				t.Data = []byte(hex.Dump(t.Data))
			}
		}

	}

	ct := currentTab()
	if ct != nil && !newfile && len(ct.Data) == 0 {
		closeCurrentTab()
	}

	t.swin = gtk.NewScrolledWindow(nil, nil)
	t.swin.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	t.swin.SetShadowType(gtk.SHADOW_IN)

	if issetLanguage(t.Language) {
		t.sourcebuffer = gsv.NewSourceBufferWithLanguage(gsv.SourceLanguageManagerGetDefault().GetLanguage(t.Language))
	} else {
		t.sourcebuffer = gsv.NewSourceBuffer()
	}

	t.sourceview = gsv.NewSourceViewWithBuffer(t.sourcebuffer)
	t.sourceview.SetHighlightCurrentLine(conf.TextView.LineHightlight)
	t.sourceview.ModifyFontEasy(conf.TextView.Font)

	if t.Encoding != "binary" {
		t.sourceview.SetShowLineNumbers(conf.TextView.LineNumbers)
		t.sourceview.SetTabWidth(uint(conf.TextView.IndentWidth))
		t.sourceview.SetInsertSpacesInsteadOfTabs(conf.TextView.IndentSpace)

		if conf.TextView.WordWrap {
			t.sourceview.SetWrapMode(gtk.WRAP_WORD_CHAR)
		}
	}

	t.DragAndDrop()

	var start gtk.TextIter
	t.sourcebuffer.GetStartIter(&start)
	t.sourcebuffer.BeginNotUndoableAction()
	t.sourcebuffer.Insert(&start, string(t.Data))
	t.sourcebuffer.EndNotUndoableAction()

	t.sourcebuffer.Connect("changed", t.onchange)

	t.swin.Add(t.sourceview)

	t.label = gtk.NewLabel(path.Base(filename))
	t.label.SetTooltipText(filename)

	if newfile {
		t.SetTabFGColor(conf.Tabs.FGNew)
	} else {
		t.SetTabFGColor(conf.Tabs.FGNormal)
	}

	n := notebook.AppendPage(t.swin, t.label)
	notebook.ShowAll()
	notebook.SetCurrentPage(n)
	t.sourceview.GrabFocus()

	tabs = append(tabs, t)

	// homogenousTabs()
}

func (t *Tab) DetectEncoding() string {
	contentType := strings.Split(http.DetectContentType(t.Data), ";")
	if !strings.HasPrefix(contentType[0], "text") {
		return "binary"
	}

	if len(contentType) >= 1 {
		return "utf-8"
	}

	enc := strings.Split(contentType[1], "=")
	if len(enc) > 1 {
		return enc[1]
	}

	return "utf-8"
	// return
}

func (t *Tab) Hex() string {
	// t.File.Seek(0, 0)
	// var hexdata []string
	// var b = make([]byte, 16)
	// var err error
	// var n int
	// for err == nil {
	// 	n, err = t.File.Read(b)
	// 	b = b[:n]

	// 	var line []string
	// 	for n := range b {
	// 		if n%2 == 0 {
	// 			line = append(line, fmt.Sprintf("%02x%02x", b[n], b[n+1]))
	// 		}
	// 	}
	// 	hexdata = append(hexdata, strings.Join(line, " "))
	// }

	// return strings.Join(hexdata, "\n")

	// t.File.Seek(0, 0)
	var hexdata []string
	var b = make([]byte, 16)
	var err error
	var n int
	reader := bytes.NewReader(t.Data)
	for err == nil {
		n, err = reader.Read(b)
		b = b[:n]

		hexdata = append(hexdata, fmt.Sprintf("% 02x", b))
	}

	return strings.Join(hexdata, "\n")
}

func (t *Tab) DetectLanguage() string {
	if len(languages) == 0 {
		return ""
	}

	ext := path.Ext(t.Filename)
	if len(ext) > 0 {
		ext = ext[1:]
	}
	if issetLanguage(ext) {
		return ext
	}

	line := string(bytes.SplitN(t.Data[:64], []byte("\n"), 2)[0])
	_line := strings.Split(line, " ")
	if issetLanguage(_line[len(_line)-1]) {
		return _line[len(_line)-1]
	}

	_, f := path.Split(_line[0])
	if issetLanguage(f) {
		return f
	}

	maybexml := strings.Trim(_line[0], "<?#")
	if issetLanguage(maybexml) {
		return maybexml
	}

	return strings.ToLower(gsv.NewSourceLanguageManager().GuessLanguage(t.Filename, "").GetName())
}

func (t *Tab) DragAndDrop() {
	t.sourceview.DragDestSet(gtk.DEST_DEFAULT_MOTION|gtk.DEST_DEFAULT_HIGHLIGHT|gtk.DEST_DEFAULT_DROP, targets, gdk.ACTION_COPY)
	t.sourceview.DragDestAddUriTargets()
	t.sourceview.Connect("drag-data-received", DnDHandler)
}

func DnDHandler(ctx *glib.CallbackContext) {
	sdata := gtk.NewSelectionDataFromNative(unsafe.Pointer(ctx.Args(3)))
	if sdata != nil {
		a := (*[2048]uint8)(sdata.GetData())
		files := strings.Split(string(a[:sdata.GetLength()-1]), "\n")
		for _, filename := range files {
			filename = filename[:len(filename)-1]

			u, err := url.Parse(filename)
			if err != nil {
				fmt.Println("failed parse path to file", err)
				continue
			}

			if u.Scheme == "file" {
				filename = u.Path
			} else {
				filename = path.Join(gvfsPath, fmt.Sprintf("%s:host=%s", u.Scheme, u.Host), u.Path)
			}

			NewTab(filename)
		}
	}
}

func (t *Tab) onchange() {
	t.Data = t.GetText()
	t.SetTabFGColor(conf.Tabs.FGModified)
}

func (t *Tab) SetTabFGColor(col [3]int) {
	r := uint16(math.Pow(float64(col[0]), 2))
	g := uint16(math.Pow(float64(col[1]), 2))
	b := uint16(math.Pow(float64(col[2]), 2))

	color := gdk.NewColorRGB(r, g, b)
	t.label.ModifyFG(gtk.STATE_NORMAL, color)
	t.label.ModifyFG(gtk.STATE_PRELIGHT, color)
	t.label.ModifyFG(gtk.STATE_SELECTED, color)
	t.label.ModifyFG(gtk.STATE_ACTIVE, color)
}

func (t *Tab) Save() {
	log.Println("SAVE")

	var text []byte
	if t.Encoding == "binary" {
		log.Println("sorry, binary data save not yet done")
	} else if t.ReadOnly {
		log.Println("sorry, file is open readonly mode")
		return
	} else if t.Encoding == "ASCII" || t.Encoding == "UTF-8" {
		text = t.GetText()
	} else {
		text, err = changeEncoding(t.GetText(), t.Encoding, "utf-8")
		if err != nil {
			log.Println("failed restore encoding, save failed,", err)
			return
		}
	}

	// log.Println(string(text), t.Encoding)

	if t.File == nil {
		var err error
		t.File, err = os.OpenFile(t.Filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Println(err)
			return
		}
	}

	t.File.Seek(0, 0)
	n, err := t.File.Write(text)
	if err != nil {
		log.Println("failed write data", err)
		return
	}
	t.File.Truncate(int64(n))
	t.SetTabFGColor(conf.Tabs.FGNormal)
}

func (t *Tab) GetText() []byte {
	var start gtk.TextIter
	var end gtk.TextIter

	t.sourcebuffer.GetStartIter(&start)
	t.sourcebuffer.GetEndIter(&end)
	return []byte(t.sourcebuffer.GetText(&start, &end, true))
}

func (t *Tab) Find(substr string) {
	if t.tagfind != nil {
		tabletag := t.sourcebuffer.GetTagTable()
		tabletag.Remove(t.tagfind)
	}

	if len(substr) == 0 {
		t.tagfind = nil
		return
	}

	t.tagfind = t.sourcebuffer.CreateTag("find", map[string]string{"background": "#999999"})

	regexp.MustCompile("").ReplaceAllString(substr, "")
	//
	// search with regexp
	reg, err := regexp.Compile("(?im)" + substr)
	if err != nil {
		t.tagfind = nil
		log.Println("invalid regexp query", err)
		return
	}

	lensubstr := len([]rune(ClearMeta(substr)))
	if lensubstr == 0 {
		return
	}
	// log.Println(ClearMeta(substr))

	data := t.GetText()
	matches := reg.FindAllIndex(data, conf.Search.MaxItems)
	for n, index := range matches {
		i := utf8.RuneCount(data[:index[0]])
		size := utf8.RuneCount(data[index[0]:index[1]])
		t.Highlight(i, size)
		if n == 0 {
			t.Scroll(i)
		}
	}

	// //
	// // search plane text
	// runeText := []rune(string(t.GetText()))

	// var c int
	// for i := 0; i < len(runeText); i++ {
	// 	if string(runeText[i:i+lensubstr]) == substr {
	// 		t.Highlight(i, lensubstr)
	// 		if c == 0 {
	// 			t.Scroll(i)
	// 		}
	// 		c++
	// 		if c > conf.Search.MaxItems {
	// 			break
	// 		}
	// 	}
	// }
}

func (t *Tab) Highlight(index, size int) {
	var start gtk.TextIter
	var end gtk.TextIter
	t.sourcebuffer.GetIterAtOffset(&start, index)
	t.sourcebuffer.GetIterAtOffset(&end, index+size)
	t.sourcebuffer.ApplyTag(t.tagfind, &start, &end)
}

func (t *Tab) Scroll(index int) {
	var start gtk.TextIter
	t.sourcebuffer.GetIterAtOffset(&start, index)
	t.sourceview.ScrollToIter(&start, 0, false, 0, 0)
}
