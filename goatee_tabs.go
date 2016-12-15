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

	if !newfile {
		t.File, err = os.Open(filename)
		if err != nil {
			log.Println("failed open file", filename, err)
			return
		}
		defer t.File.Close()

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

	n := ui.notebook.AppendPage(t.swin, t.label)
	ui.notebook.ShowAll()
	ui.notebook.SetCurrentPage(n)
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
	var text []byte
	if t.Encoding == "binary" {
		log.Println("sorry, binary data save not yet done")
	} else if t.ReadOnly {
		log.Println("sorry, file is open readonly mode")
		return
	} else if t.Encoding == "ASCII" || t.Encoding == "UTF-8" {
		text = []byte(t.GetText())
	} else {
		text, err = changeEncoding([]byte(t.GetText()), t.Encoding, "utf-8")
		if err != nil {
			log.Println("failed restore encoding, save failed,", err)
			return
		}
	}

	if err := ioutil.WriteFile(t.Filename, text, 0644); err != nil {
		log.Println("failed save file,", err)
		return
	}

	t.SetTabFGColor(conf.Tabs.FGNormal)
}

func (t *Tab) GetText() string {
	var start gtk.TextIter
	var end gtk.TextIter

	t.sourcebuffer.GetStartIter(&start)
	t.sourcebuffer.GetEndIter(&end)
	return t.sourcebuffer.GetText(&start, &end, true)
}

func (t *Tab) Find(substr string) {
	if t.tagfind != nil {
		tabletag := t.sourcebuffer.GetTagTable()
		tabletag.Remove(t.tagfind)
		tabletag.Remove(t.tagfindCurrent)
	}

	lensubstr := len([]rune(substr))
	if lensubstr == 0 {
		t.tagfind = nil
		t.tagfindCurrent = nil
		return
	}

	text := t.GetText()

	t.findindex = [][]int{}

	t.tagfind = t.sourcebuffer.CreateTag("find", map[string]string{"background": "#999999"})
	t.tagfindCurrent = t.sourcebuffer.CreateTag("findCurr", map[string]string{"background": "#eeaa00"})

	if ui.footer.caseBtn.GetActive() {
		substr = strings.ToLower(substr)
	}

	if !ui.footer.regBtn.GetActive() {
		substr = regexp.QuoteMeta(substr)
	}

	reg, err := regexp.Compile("(?m)" + substr)
	if err != nil {
		log.Println("invalid regexp query", err)
		return
	}

	t.findindex = reg.FindAllIndex([]byte(text), -1)
	// t.findindex = reg.FindAllStringIndex(text, -1)

	data := []byte(text)
	for i, index := range t.findindex {
		index[0] = utf8.RuneCount(data[:index[0]])
		index[1] = utf8.RuneCount(data[:index[1]])
		if i == 0 {
			t.Highlight(index, true)
		} else {
			t.Highlight(index, false)
		}
	}

	// if ui.footer.regBtn.GetActive() {
	// 	//
	// 	// search with regexp

	// 	//create regexp expresion
	// 	reg, err := regexp.Compile("(?m)" + substr)
	// 	if err != nil {
	// 		// t.tagfind = nil
	// 		// t.tagfindCurrent = nil
	// 		log.Println("invalid regexp query", err)
	// 		return
	// 	}

	// 	data := []byte(t.GetText())
	// 	t.findindex = reg.FindAllIndex(data, conf.Search.MaxItems)
	// 	for n, index := range t.findindex {
	// 		offset := utf8.RuneCount(data[:index[0]])
	// 		size := utf8.RuneCount(data[index[0]:index[1]])
	// 		index := []int{offset, offset + size}

	// 		if n == 0 {
	// 			t.Highlight(index, true)
	// 			t.findindexCurrent = n
	// 		} else {
	// 			t.Highlight(index, false)
	// 		}
	// 	}
	// } else {
	// 	// //
	// 	// search plane text

	// 	var runeText []rune
	// 	if ui.footer.caseBtn.GetActive() {
	// 		runeText = []rune(strings.ToLower(t.GetText()))
	// 		substr = strings.ToLower(substr)
	// 	} else {
	// 		runeText = []rune(t.GetText())
	// 	}

	// 	var n int
	// 	for i := 0; i < len(runeText); i++ {
	// 		if i+lensubstr > len(runeText) {
	// 			continue
	// 		}
	// 		if string(runeText[i:i+lensubstr]) == substr {
	// 			index := []int{i, i + lensubstr}
	// 			t.findindex = append(t.findindex, index)

	// 			if n == 0 {
	// 				t.Highlight(index, true)
	// 				t.findindexCurrent = n
	// 			} else {
	// 				t.Highlight(index, false)
	// 			}

	// 			n++
	// 			if n > conf.Search.MaxItems {
	// 				break
	// 			}
	// 		}
	// 	}
	// }
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
	t.sourceview.ScrollToIter(&iter, 0, false, 0, 0)
}

func (t *Tab) Replace(all bool) {
	text := t.GetText()
	findtext := ui.footer.findEntry.GetText()
	repltext := ui.footer.replEntry.GetText()

	var n = 1
	if all {
		n = -1
	}

	if ui.footer.caseBtn.GetActive() {
		findtext = strings.ToLower(findtext)
	}

	if ui.footer.regBtn.GetActive() {
		reg, err := regexp.Compile("(?m)" + findtext)
		if err != nil {
			log.Println("failed compile regexp", err)
			return
		}
		text = reg.ReplaceAllString(text, repltext)
	} else {
		text = strings.Replace(text, findtext, repltext, n)
	}

	t.sourcebuffer.SetText(text)

	t.Find(ui.footer.findEntry.GetText())
}
