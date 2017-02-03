package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
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
	"github.com/saintfish/chardet"

	gsv "github.com/mattn/go-gtk/gtksourceview"
)

type Tab struct {
	Filename string
	File     *os.File
	Encoding string
	Language string
	ReadOnly bool
	// Empty    bool

	tabbox   *gtk.HBox
	label    *gtk.Label
	closeBtn *gtk.Button

	swin         *gtk.ScrolledWindow
	sourceview   *gsv.SourceView
	sourcebuffer *gsv.SourceBuffer

	findindex        [][]int
	findindexCurrent int
	tagfind          *gtk.TextTag
	tagfindCurrent   *gtk.TextTag
}

func (ui *UI) NewTab(filename string) {
	if len(filename) > 0 && ui.TabsContains(filename) {
		return
	}

	t := &Tab{
		Encoding: CHARSET_UTF8,
	}

	if filename == "" {
		filename = fmt.Sprintf("new%d", newtabiter)
		newtabiter++
	} else {
		t.Filename = filename
	}

	if len(t.Filename) > 0 {
		ct := ui.GetCurrentTab()
		if ct != nil && len(ct.Filename) == 0 {
			ct.Close()
			// ui.CloseCurrentTab()
		}
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

	t.swin.Add(t.sourceview)

	t.label = gtk.NewLabel(path.Base(filename))
	t.label.SetTooltipText(filename)

	if len(t.Filename) == 0 {
		t.SetTabFGColor(conf.Tabs.FGNew)
	} else {
		t.SetTabFGColor(conf.Tabs.FGNormal)
	}

	t.tabbox = gtk.NewHBox(false, 0)
	t.tabbox.PackStart(t.label, true, true, 0)

	if conf.Tabs.CloseBtns {
		t.closeBtn = gtk.NewButton()
		t.closeBtn.Add(gtk.NewImageFromStock(gtk.STOCK_CLOSE, gtk.ICON_SIZE_BUTTON))
		t.closeBtn.SetRelief(gtk.RELIEF_NONE)
		t.closeBtn.SetSizeRequest(conf.Tabs.Height, conf.Tabs.Height)
		t.closeBtn.Clicked(t.Close)
		t.tabbox.PackEnd(t.closeBtn, false, false, 0)
	}

	t.tabbox.ShowAll()

	if len(t.Filename) > 0 {
		text, err := t.ReadFile(filename)
		if err != nil {
			errorMessage(err)
			log.Println(err)
			return
		}

		t.sourcebuffer.BeginNotUndoableAction()
		t.sourcebuffer.SetText(text)
		t.sourcebuffer.EndNotUndoableAction()
	}

	if issetLanguage(t.Language) {
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

	ui.tabs = append(ui.tabs, t)

	n := ui.notebook.AppendPage(t.swin, t.tabbox)
	ui.notebook.ShowAll()
	ui.notebook.SetCurrentPage(n)
	t.sourceview.GrabFocus()
}

func (t *Tab) Close() {
	n := ui.notebook.PageNum(t.swin)

	ui.notebook.RemovePage(ui.tabs[n].swin, n)
	ui.tabs[n].File.Close()
	ui.tabs = append(ui.tabs[:n], ui.tabs[n+1:]...)

	// ui.CloseTab(n)
}

func (t *Tab) ReadFile(filename string) (string, error) {
	var err error
	t.File, err = os.Open(filename)
	if err != nil {
		err := fmt.Errorf("failed open file  `%s`, %s", filename, err)
		return "", err
	}
	defer t.File.Close()

	data, err := ioutil.ReadAll(t.File)
	if err != nil {
		err := fmt.Errorf("failed read file  `%s`, %s", filename, err)
		return "", err
	}

	if len(data) > 0 {
		t.Encoding, err = t.DetectEncoding(data)
		log.Println(t.Encoding, err)
		if err != nil {
			t.Encoding = CHARSET_BINARY
		}

		if t.Encoding != CHARSET_UTF8 && t.Encoding != CHARSET_BINARY {
			newdata, err := changeEncoding(data, CHARSET_UTF8, t.Encoding)
			// log.Println(t.Encoding, err)
			if err != nil {
				errorMessage(err)
				t.Encoding = CHARSET_BINARY
			} else {
				data = newdata
			}

		}

		if t.Encoding != CHARSET_BINARY {
			t.Language = t.DetectLanguage(data)
			return string(data), nil
		}

		if issetLanguage("hex") {
			t.Language = "hex"
		}

		return bytetohex(bytes.NewReader(data)), nil
	}
	return "", nil
}

const CHARSET_BINARY = "binary"
const CHARSET_UTF8 = "utf-8"

func (t *Tab) DetectEncoding(data []byte) (string, error) {
	contentType := strings.Split(http.DetectContentType(data), ";")
	if len(contentType) != 2 {
		c, err := t.DetectChardet(data)
		if err != nil {
			return "", errors.New("failed split content type amd detect charset")
		}
		return c, nil
	}

	charset := strings.Split(contentType[1], "=")
	if len(charset) != 2 {
		return "", errors.New("failed split charset")
	}

	if charset[1] == CHARSET_UTF8 && !utf8.Valid(data) {
		return t.DetectChardet(data)
		// r, err := chardet.NewTextDetector().DetectBest(data)
		// log.Println(r)
		// if err != nil || r.Confidence < 30 {
		// 	return "", errors.New("failed detect charset")
		// }
		// return r.Charset, nil

	}

	return charset[1], nil

	// r, err := chardet.NewTextDetector().DetectBest(data)
	// log.Println(r)
	// if err != nil || r.Confidence < 30 {
	// 	return CHARSET_BINARY
	// }

	// return r.Charset
}

func (t *Tab) DetectChardet(data []byte) (string, error) {
	r, err := chardet.NewTextDetector().DetectBest(data)
	if err != nil || r.Confidence < 30 {
		return "", errors.New("failed detect charset with chardet")
	}
	return r.Charset, nil
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

	if strings.HasSuffix(t.Filename, "rc") {
		return "sh"
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

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
			if issetLanguage("yaml") {
				return "yaml"
			}
			if issetLanguage("desktop") {
				return "desktop"
			}
		}

		if line[0] == ';' {
			if issetLanguage("ini") {
				return "ini"
			}
		}
	}
	if ext == ".conf" || ext == ".cfg" {
		if issetLanguage("ini") {
			return "ini"
		}
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

			ui.NewTab(filename)
		}
	}
}

func (t *Tab) onchange() {
	// t.Data = t.GetText()
	t.SetTabFGColor(conf.Tabs.FGModified)
	// t.Empty = false
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

func (t *Tab) ClearTags() {
	tabletag := t.sourcebuffer.GetTagTable()

	if tag := tabletag.Lookup("find"); tag != nil {
		tabletag.Remove(tag)
	}

	if tag := tabletag.Lookup("findCurr"); tag != nil {
		tabletag.Remove(tag)
	}
}

func (t *Tab) Find() {
	t.ClearTags()

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

// func (t *Tab) RemoveTag(name string) {
// 	tagtable := t.sourcebuffer.GetTagTable()
// 	if tag := tagtable.Lookup(name); tag != nil {
// 		tagtable.Remove(tag)
// 	}
// }

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
