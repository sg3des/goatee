package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
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
	Encoding string
	Language string
	ReadOnly bool

	//color        *gdk.Color
	swin         *gtk.ScrolledWindow
	label        *gtk.Label
	sourcebuffer *gsv.SourceBuffer
	sourceview   *gsv.SourceView

	findindex        [][]int
	findindexCurrent int
	tagfind          *gtk.TextTag
	tagfindCurrent   *gtk.TextTag
}

func NewTab(filename string) {
	var newfile bool
	var err error

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

	if ct := currentTab(); ct != nil && newfile {
		closeCurrentTab()
	}

	t.swin = gtk.NewScrolledWindow(nil, nil)
	t.swin.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	t.swin.SetShadowType(gtk.SHADOW_IN)

	t.sourcebuffer = gsv.NewSourceBuffer()

	t.sourcebuffer.SetStyleScheme(gsv.NewSourceStyleSchemeManager().GetScheme("classic"))

	t.sourceview = gsv.NewSourceViewWithBuffer(t.sourcebuffer)
	t.sourceview.SetHighlightCurrentLine(conf.TextView.LineHightlight)
	t.sourceview.ModifyFontEasy(conf.TextView.Font)
	t.sourceview.SetShowLineNumbers(conf.TextView.LineNumbers)

	t.DragAndDrop()

	// var start gtk.TextIter
	// t.sourcebuffer.GetStartIter(&start)
	// t.sourcebuffer.BeginNotUndoableAction()
	// t.sourcebuffer.Insert(&start, string(t.Data))
	// t.sourcebuffer.EndNotUndoableAction()

	t.swin.Add(t.sourceview)

	t.label = gtk.NewLabel(path.Base(filename))
	t.label.SetTooltipText(filename)

	if newfile {
		t.SetTabFGColor(conf.Tabs.FGNew)
	} else {
		t.SetTabFGColor(conf.Tabs.FGNormal)
	}

	n := ui.notebook.AppendPage(t.swin, t.label)
	ui.notebook.ShowAll()
	ui.notebook.SetCurrentPage(n)
	t.sourceview.GrabFocus()

	// text
	var text string
	if !newfile {
		t.File, err = os.Open(filename)
		if err != nil {
			err := fmt.Errorf("failed open file  `%s`, %s", filename, err)
			errorMessage(err)
			log.Println(err)
			return
		}
		defer t.File.Close()

		stat, err := t.File.Stat()
		if err != nil {
			err := fmt.Errorf("failed get stat of file  `%s`, %s", filename, err)
			errorMessage(err)
			log.Println(err)
			return
		}

		data, err := ioutil.ReadAll(t.File)
		if err != nil {
			err := fmt.Errorf("failed read file  `%s`, %s", filename, err)
			errorMessage(err)
			log.Println(err)
			return
		}

		if stat.Size() > 0 {
			t.Encoding = t.DetectEncoding(data)

			if t.Encoding != "utf-8" && t.Encoding != "binary" {
				data, err = changeEncoding(data, "utf-8", t.Encoding)
				if err != nil {
					err := fmt.Errorf("failed change encding, %s", err)
					errorMessage(err)
					log.Println(err)
					return
				}
			}

			if t.Encoding != "binary" {
				t.Language = t.DetectLanguage(data)
				text = string(data)
			} else {
				text = bytetohex(bytes.NewReader(data))
				if issetLanguage("hex") {
					t.Language = "hex"
				}
			}
		}
	}
	t.sourcebuffer.BeginNotUndoableAction()
	t.sourcebuffer.SetText(text)
	t.sourcebuffer.EndNotUndoableAction()

	if issetLanguage(t.Language) {
		// t.sourcebuffer = gsv.NewSourceBufferWithLanguage()
		t.sourcebuffer.SetLanguage(gsv.SourceLanguageManagerGetDefault().GetLanguage(t.Language))
	}

	if t.Encoding != "binary" {
		t.sourceview.SetTabWidth(uint(conf.TextView.IndentWidth))
		t.sourceview.SetInsertSpacesInsteadOfTabs(conf.TextView.IndentSpace)

		if conf.TextView.WordWrap {
			t.sourceview.SetWrapMode(gtk.WRAP_WORD_CHAR)
		}
	}

	t.sourcebuffer.Connect("changed", t.onchange)

	tabs = append(tabs, t)
}

func (t *Tab) OnScroll() {
	log.Println("OnScroll")
}

func (t *Tab) DetectEncoding(data []byte) string {
	contentType := strings.Split(http.DetectContentType(data), ";")
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
}

// func (t *Tab) Hex() string {
// 	_, err := t.File.Seek(0, 0)
// 	if err != nil {
// 		log.Println("failed reset offset", err)
// 	}

// 	return bytetohex(t.File)
// }

func (t *Tab) DetectLanguage(data []byte) string {
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

	size := 64
	if size > len(data) {
		size = len(data)
	}
	line := string(bytes.SplitN(data[:size], []byte("\n"), 2)[0])
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

	// log.Println(strings.LastIndex(t.Filename, "rc"))

	if strings.HasSuffix(t.Filename, "rc") {
		return "sh"
	}

	return strings.ToLower(gsv.NewSourceLanguageManager().GuessLanguage(t.Filename, "").GetName())
}

func (t *Tab) DragAndDrop() {
	// dndtargets := []gtk.TargetEntry{
	// 	{"text/uri-list", 0, 0},
	// 	{"STRING", 0, 1},
	// 	{"text/plain", 0, 2},
	// }

	// t.swin.DragDestSet(gtk.DEST_DEFAULT_HIGHLIGHT|gtk.DEST_DEFAULT_DROP|gtk.DEST_DEFAULT_MOTION, dndtargets, gdk.ACTION_COPY)
	t.sourceview.DragDestAddUriTargets()
	t.sourceview.Connect("drag-data-received", t.DnDHandler)
}

func (t *Tab) DnDHandler(ctx *glib.CallbackContext) {

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

			if len(u.Scheme) == 0 {
				return
			} else if u.Scheme == "file" {
				filename = u.Path
			} else {
				filename = path.Join(gvfsPath, fmt.Sprintf("%s:host=%s", u.Scheme, u.Host), u.Path)
			}

			NewTab(filename)
		}
	}
}

func (t *Tab) onchange() {
	// t.Data = t.GetText()
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
	var err error
	var data []byte
	if t.Encoding == "binary" {

		data, err = hextobyte(t.GetText(false))
		if err != nil {
			err := fmt.Errorf("failed decode hex, %s", err)
			errorMessage(err)
			log.Println(err)
			return
		}

	} else if t.ReadOnly {

		// log.Println("sorry, file is open readonly mode")
		err := fmt.Errorf("file %s is read only", t.Filename)
		errorMessage(err)
		log.Println(err)
		return

	} else if t.Encoding == "ASCII" || t.Encoding == "UTF-8" {

		data = []byte(t.GetText(true))

	} else {

		data, err = changeEncoding([]byte(t.GetText(true)), t.Encoding, "utf-8")
		if err != nil {
			err := fmt.Errorf("failed restore encoding, save failed, %s", err)
			errorMessage(err)
			log.Println(err)
			return
		}

	}

	if err := ioutil.WriteFile(t.Filename, data, 0644); err != nil {
		err := fmt.Errorf("failed save file `%s`, %s", t.Filename, err)
		errorMessage(err)
		log.Println(err)
		return
	}

	t.SetTabFGColor(conf.Tabs.FGNormal)
}

func (t *Tab) GetText(hiddenChars bool) string {
	var start gtk.TextIter
	var end gtk.TextIter

	t.sourcebuffer.GetStartIter(&start)
	t.sourcebuffer.GetEndIter(&end)
	return t.sourcebuffer.GetText(&start, &end, hiddenChars)
}

func (t *Tab) Find() {
	if t.tagfind != nil {
		tabletag := t.sourcebuffer.GetTagTable()
		tabletag.Remove(t.tagfind)
		tabletag.Remove(t.tagfindCurrent)
	}

	find := ui.footer.findEntry.GetText()
	if len(find) == 0 {
		t.tagfind = nil
		t.tagfindCurrent = nil
		return
	}

	flags := "ms"
	if ui.footer.caseBtn.GetActive() {
		flags += "i"
	}

	if !ui.footer.regBtn.GetActive() {
		find = regexp.QuoteMeta(find)
	}

	if t.Encoding == "binary" {
		find = regexp.MustCompile("[ \n\r]+").ReplaceAllString(find, "")
		find = regexp.MustCompile("(?i)([0-9a-z]{2})").ReplaceAllString(find, "$1[ \r\n]*")
	}

	text := t.GetText(true)

	expr := fmt.Sprintf("(?%s)%s", flags, find)
	reg, err := regexp.Compile(expr)
	if err != nil {
		log.Println("invalid search query,", err)
		return
	}
	// log.Println(expr)
	t.findindex = reg.FindAllStringIndex(text, conf.Search.MaxItems)

	t.tagfind = t.sourcebuffer.CreateTag("find", map[string]string{"background": "#999999"})
	t.tagfindCurrent = t.sourcebuffer.CreateTag("findCurr", map[string]string{"background": "#eeaa00"})

	for i, index := range t.findindex {
		data := []byte(text)
		if t.Encoding != "binary" {
			index[0] = utf8.RuneCount(data[:index[0]])
			index[1] = utf8.RuneCount(data[:index[1]])
		}
		if i == 0 {
			t.Highlight(index, true)
		} else {
			t.Highlight(index, false)
		}
	}
}

func (t *Tab) FindNext(next bool) {
	if len(t.findindex) < 2 {
		return
	}

	t.Highlight(t.findindex[t.findindexCurrent], false)

	if next {
		t.findindexCurrent++
	} else {
		t.findindexCurrent--
	}

	if t.findindexCurrent >= len(t.findindex) {
		t.findindexCurrent = 0
	}
	if t.findindexCurrent < 0 {
		t.findindexCurrent = len(t.findindex) - 1
	}

	t.Highlight(t.findindex[t.findindexCurrent], true)
}

func (t *Tab) Highlight(index []int, current bool) {
	var start gtk.TextIter
	var end gtk.TextIter
	t.sourcebuffer.GetIterAtOffset(&start, index[0])
	t.sourcebuffer.GetIterAtOffset(&end, index[1])

	if current {
		t.sourcebuffer.RemoveTag(t.tagfind, &start, &end)
		t.sourcebuffer.ApplyTag(t.tagfindCurrent, &start, &end)
		t.Scroll(start)
	} else {
		t.sourcebuffer.RemoveTag(t.tagfindCurrent, &start, &end)
		t.sourcebuffer.ApplyTag(t.tagfind, &start, &end)
	}
}

func (t *Tab) Scroll(iter gtk.TextIter) {
	// log.Println(iter.GetOffset())
	t.sourceview.ScrollToIter(&iter, 0, false, 0, 0)
}

func (t *Tab) Replace(all bool) {
	var n = 1
	if all {
		n = -1
	}

	if t.Encoding != "binary" {
		t.replaceInText(n)
	} else {
		t.replaceInHex(n)
	}
}

func (t *Tab) replaceInText(n int) {
	findtext := ui.footer.findEntry.GetText()
	repltext := ui.footer.replEntry.GetText()

	if ui.footer.caseBtn.GetActive() {
		findtext = strings.ToLower(findtext)
	}

	var text string
	if ui.footer.regBtn.GetActive() {
		reg, err := regexp.Compile("(?m)" + findtext)
		if err != nil {
			log.Println("failed compile regexp", err)
			return
		}
		log.Println("regexp always replace all occurrences")
		text = reg.ReplaceAllString(t.GetText(true), repltext)
	} else {
		text = strings.Replace(t.GetText(true), findtext, repltext, n)
	}

	t.sourcebuffer.SetText(text)

	t.Find()
}

func (t *Tab) replaceInHex(n int) {
	find, err := hextobyte(ui.footer.findEntry.GetText())
	if err != nil {
		log.Println("invalid hex string", err)
		return
	}
	repl, err := hextobyte(ui.footer.replEntry.GetText())
	if err != nil {
		log.Println("invalid hex string", err)
		return
	}

	data, err := hextobyte(t.GetText(false))
	if err != nil {
		log.Println("invalid hex string", err)
		return
	}

	data = bytes.Replace(data, find, repl, n)

	t.sourcebuffer.SetText(string(data))

	text := bytetohex(bytes.NewReader(data))
	t.sourcebuffer.SetText(text)
	t.Find()
}

func hextobyte(hexstr string) ([]byte, error) {
	hexstr = regexp.MustCompile("(?m)[ \n\r]").ReplaceAllString(hexstr, "")
	return hex.DecodeString(hexstr)
}

func bytetohex(r io.Reader) string {
	var dump []string
	var line = make([]byte, conf.Hex.BytesInLine)
	for {
		n, err := r.Read(line)
		if err != nil && err != io.EOF {
			err := fmt.Errorf("failed read file %s", err)
			errorMessage(err)
			log.Println(err)
			break
		}

		line = line[:n]
		if err == io.EOF {
			break
		}

		dump = append(dump, fmt.Sprintf("% x", line))
	}

	return strings.Join(dump, "\n")
}
