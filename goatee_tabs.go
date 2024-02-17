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
	"net/http"
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

	iconv "github.com/djimenez/iconv-go"
	"github.com/saintfish/chardet"
)

type Tab struct {
	Filename string
	File     *os.File
	Encoding string
	Language string
	ReadOnly bool
	Dirty    bool

	eventbox *gtk.EventBox
	tab      *gtk.HBox
	label    *gtk.Label
	closeBtn *gtk.Button

	swin         *gtk.ScrolledWindow
	sourceview   *gsv.SourceView
	sourcebuffer *gsv.SourceBuffer

	cursorPos gtk.TextIter

	find             string
	findtext         string
	findindex        [][]int
	findindexCurrent int
	findoffset       int
	findwrap         bool
	tagfind          *gtk.TextTag
	tagfindCurrent   *gtk.TextTag
}

func NewTab(filename string) (t *Tab) {
	if len(filename) > 0 {
		filename = resolveFilename(filename)
	}

	if len(filename) > 0 {
		var ok bool
		var n int

		//reload if this file already open
		if t, n, ok = ui.LookupTab(filename); ok {
			ui.notebook.SetCurrentPage(n)

			text, err := t.ReadFile(filename)
			if err != nil {
				errorMessage(err)
				log.Println(err)
				return
			}
			t.sourcebuffer.SetText(text)
			t.Dirty = false
			t.SetTabFGColor(conf.Tabs.FGNormal)
			//TODO: reopen
			return nil
		}
	}

	t = &Tab{
		Encoding: CHARSET_UTF8,
		Language: "sh",
	}

	if len(filename) == 0 {
		filename = fmt.Sprintf("new%d", newtabiter)
		newtabiter++
	} else {
		t.Filename = filename
	}

	if len(t.Filename) > 0 {
		ct := ui.GetCurrentTab()
		if ct != nil && len(ct.Filename) == 0 && !ct.Dirty {
			ct.Close()
		}
	}

	t.swin = gtk.NewScrolledWindow(nil, nil)
	t.swin.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	t.swin.SetShadowType(gtk.SHADOW_IN)

	t.sourcebuffer = gsv.NewSourceBuffer()
	t.sourceview = gsv.NewSourceViewWithBuffer(t.sourcebuffer)

	t.DragAndDrop()

	t.swin.Add(t.sourceview)

	t.label = gtk.NewLabel(path.Base(filename))
	t.label.SetTooltipText(filename)

	t.tab = gtk.NewHBox(false, 0)
	t.tab.PackStart(t.label, true, true, 0)

	if len(t.Filename) > 0 {

		stat, err := os.Stat(filename)
		if err == nil && !stat.IsDir() {

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
	}

	if issetLanguage(t.Language) {
		t.sourcebuffer.SetLanguage(langManager.GetLanguage(t.Language))
	}

	t.ApplyConf()

	// t.tab.ShowAll()

	t.eventbox = gtk.NewEventBox()
	t.eventbox.Connect("button_press_event", t.onTabPress)
	t.eventbox.Add(t.tab)
	t.eventbox.ShowAll()

	t.sourcebuffer.Connect("changed", t.onchange)
	t.sourcebuffer.Connect("notify::cursor-position", t.onMoveCursor)

	return t
}

func (t *Tab) ApplyConf() {
	t.sourcebuffer.SetStyleScheme(conf.schemeManager.GetScheme(conf.TextView.StyleScheme))

	t.sourceview.SetHighlightCurrentLine(conf.TextView.LineHightlight)
	t.sourceview.ModifyFontEasy(conf.TextView.Font)
	t.sourceview.SetShowLineNumbers(conf.TextView.LineNumbers)

	if len(t.Filename) == 0 {
		t.SetTabFGColor(conf.Tabs.FGNew)
	} else {
		t.SetTabFGColor(conf.Tabs.FGNormal)
	}

	t.tab.SetSizeRequest(-1, conf.Tabs.Height)

	if conf.Tabs.CloseBtns {
		if t.closeBtn == nil {
			t.closeBtn = gtk.NewButton()
			t.closeBtn.Add(gtk.NewImageFromStock(gtk.STOCK_CLOSE, gtk.ICON_SIZE_BUTTON))
			t.closeBtn.SetRelief(gtk.RELIEF_NONE)
			t.closeBtn.Clicked(t.close)
			t.tab.PackStart(t.closeBtn, false, false, 0)
		}

		t.closeBtn.ShowAll()
		t.closeBtn.SetSizeRequest(conf.Tabs.Height, conf.Tabs.Height)
	} else {
		if t.closeBtn != nil {
			t.closeBtn.HideAll()
		}
	}

	if t.Encoding != CHARSET_BINARY {
		t.sourceview.SetTabWidth(uint(conf.TextView.IndentWidth))
		t.sourceview.SetInsertSpacesInsteadOfTabs(conf.TextView.IndentSpace)

		if conf.TextView.WordWrap {
			t.sourceview.SetWrapMode(gtk.WRAP_WORD_CHAR)
		}
	}

}

func (t *Tab) UpdateMenuSeleted() {
	ui.NoActivate = true
	if ra, ok := ui.encodings[t.Encoding]; ok {
		ra.SetActive(true)
	}

	if ra, ok := ui.languages[t.Language]; ok {
		ra.SetActive(true)
	}
	ui.NoActivate = false
}

func (t *Tab) onTabPress(ctx *glib.CallbackContext) {
	arg := ctx.Args(0)
	event := *(**gdk.EventButton)(unsafe.Pointer(&arg))

	if event.Button == 2 {
		if _, n, ok := ui.LookupTab(t.Filename); ok {
			ui.CloseTab(n)
			return
		}
	}
}

func (t *Tab) close() {
	if _, n, ok := ui.LookupTab(t.Filename); ok {
		ui.CloseTab(n)
	}
}

func (t *Tab) Close() {
	if t.File != nil {
		t.File.Close()
	}

	t = nil
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
		if err != nil {
			t.Encoding = CHARSET_BINARY
		}

		if t.Encoding != CHARSET_UTF8 && t.Encoding != CHARSET_BINARY {
			newdata, err := t.ChangeEncoding(data, CHARSET_UTF8, t.Encoding)
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
const CHARSET_ASCII = "ascii"

func (t *Tab) DetectEncoding(data []byte) (string, error) {
	httpContentType := http.DetectContentType(data)
	if !strings.HasPrefix(httpContentType, "text") {
		return CHARSET_BINARY, nil
	}

	contentType := strings.Split(httpContentType, ";")
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
	}

	return charset[1], nil
}

func (t *Tab) DetectChardet(data []byte) (string, error) {
	// log.Println(chardet.NewTextDetector().DetectAll(data))
	r, err := chardet.NewTextDetector().DetectBest(data)
	if err != nil || r.Confidence < 30 {
		return "", errors.New("failed detect charset with chardet")
	}
	return r.Charset, nil
}

func (t *Tab) ChangeEncoding(data []byte, to, from string) ([]byte, error) {
	converter, err := iconv.NewConverter(from, to)
	if err != nil {
		return nil, fmt.Errorf("unknown charsets: `%s` `%s`, %s", to, from, err)
	}

	newdata := make([]byte, len(data)*4)
	_, n, err := converter.Convert(data, newdata)
	if err != nil {
		return nil, fmt.Errorf("failed change encoding from `%s`, %s", from, err)
	}

	return newdata[:n], nil

	// cd, err := iconv.Open(to, from)
	// if err != nil {
	// 	return nil, fmt.Errorf("unknown charsets: `%s` `%s`, %s", to, from, err)
	// }
	// defer cd.Close()

	// var outbuf = make([]byte, len(data))
	// out, _, err := cd.Conv(data, outbuf)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed change encoding from `%s`, %s", from, err)
	// }
	// return out, nil
}

func (t *Tab) ChangeCurrEncoding(from string) {
	if t == nil {
		return
	}

	// log.Println("ChangeCurrEncoding", t.Filename, t.Encoding, from)

	if t.Encoding == from {
		return
	}

	var data []byte
	var err error

	dirtyState := t.Dirty

	var tmpdata []byte
	if t.Dirty || t.File == nil {
		tmpdata = []byte(t.GetText(true))
	} else {
		tmpdata, err = ioutil.ReadFile(t.Filename)
		if err != nil {
			errorMessage(err)
			log.Println(err)
			return
		}
	}

	if t.Dirty {
		if t.Encoding == CHARSET_BINARY {
			tmpdata = regexp.MustCompile("[ \n\r]+").ReplaceAll(tmpdata, []byte{})
			data, err = hex.DecodeString(string(tmpdata))
		} else {
			data, err = t.ChangeEncoding(tmpdata, t.Encoding, CHARSET_UTF8)
		}
		if err != nil {
			errorMessage(err)
			log.Println(err)
			return
		}
	} else {
		data = tmpdata
	}

	if from == CHARSET_BINARY {
		t.Language = CHARSET_BINARY
		data = []byte(bytetohex(bytes.NewReader(data)))
	} else {
		data, err = t.ChangeEncoding(data, CHARSET_UTF8, from)
		if err != nil {
			log.Println(err)
			errorMessage(err)
			return
		}
	}

	t.Encoding = from
	if t.sourcebuffer != nil {
		t.sourcebuffer.SetText(string(data))
	}
	t.Dirty = dirtyState
}

func (t *Tab) ChangeLanguage(lang string) {
	if t == nil {
		return
	}

	if t.Language == lang {
		return
	}

	t.Language = lang
	if t.sourcebuffer != nil {
		t.sourcebuffer.SetLanguage(langManager.GetLanguage(lang))
	}
}

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

	name := gsv.NewSourceLanguageManager().GuessLanguage(t.Filename, "").GetName()
	if len(name) > 0 {
		return strings.ToLower(name)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		if line[0] == '#' {
			if issetLanguage("toml") {
				return "toml"
			}
			if issetLanguage("yaml") {
				return "yaml"
			}
			if issetLanguage("sh") {
				return "sh"
			}
			// if issetLanguage("desktop") {
			// 	return "desktop"
			// }
		}

		if line[0] == ';' {
			if issetLanguage("ini") {
				return "ini"
			}
		}

		if line[0] == '[' {
			if issetLanguage("ini") {
				return "ini"
			}
			if issetLanguage("toml") {
				return "toml"
			}
		}
	}
	// if ext == ".conf" || ext == ".cfg" {
	// 	if issetLanguage("ini") {
	// 		return "ini"
	// 	}
	// 	if issetLanguage("toml") {
	// 		return "toml"
	// 	}
	// }

	return "sh"
}

func (t *Tab) DragAndDrop() {
	t.sourceview.DragDestAddUriTargets()
	t.sourceview.Connect("drag-data-received", t.DnDHandler)
}

func (t *Tab) DnDHandler(ctx *glib.CallbackContext) {
	sdata := gtk.NewSelectionDataFromNative(unsafe.Pointer(ctx.Args(3)))
	if sdata != nil {
		a := (*[2048]uint8)(sdata.GetData())
		files := strings.Split(string(a[:sdata.GetLength()-1]), "\n")
		for _, filename := range files {
			filename = resolveFilename(filename[:len(filename)-1])
			ui.NewTab(filename)
		}
	}
}

func (t *Tab) onchange() {
	// t.Data = t.GetText()
	t.Dirty = true
	t.SetTabFGColor(conf.Tabs.FGModified)

	t.Find()
	// t.Empty = false
}

func (t *Tab) SetTabFGColor(col []int) {
	color := convertColor(col)
	t.label.ModifyFG(gtk.STATE_NORMAL, color)
	t.label.ModifyFG(gtk.STATE_PRELIGHT, color)
	t.label.ModifyFG(gtk.STATE_SELECTED, color)
	t.label.ModifyFG(gtk.STATE_ACTIVE, color)
}

func (t *Tab) Save() {
	var err error
	var data []byte
	if t.Encoding == CHARSET_BINARY {

		data, err = hextobyte(t.GetText(false))
		if err != nil {
			err := fmt.Errorf("failed decode hex, %s", err)
			errorMessage(err)
			log.Println(err)
			return
		}

	} else if t.ReadOnly {

		err := fmt.Errorf("file %s is read only", t.Filename)
		errorMessage(err)
		log.Println(err)
		return

	} else if t.Encoding == CHARSET_ASCII || t.Encoding == CHARSET_UTF8 {

		data = []byte(t.GetText(true))

	} else {

		data, err = t.ChangeEncoding([]byte(t.GetText(true)), t.Encoding, "utf-8")
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
	if t.sourcebuffer == nil {
		return ""
	}

	var start gtk.TextIter
	var end gtk.TextIter

	t.sourcebuffer.GetStartIter(&start)
	t.sourcebuffer.GetEndIter(&end)
	return t.sourcebuffer.GetText(&start, &end, hiddenChars)
}

func (t *Tab) ClearFind() {
	t.find = ""
	t.findtext = ""
	t.findindex = nil
	t.findindexCurrent = 0

	tabletag := t.sourcebuffer.GetTagTable()

	if tag := tabletag.Lookup("find"); tag != nil && tag.GTextTag != nil {
		tabletag.Remove(tag)
	}

	if tag := tabletag.Lookup("findCurr"); tag != nil && tag.GTextTag != nil {
		tabletag.Remove(tag)
	}
}

func (t *Tab) Find() {
	t.ClearFind()

	t.find = ui.footer.findEntry.GetText()
	if len(t.find) == 0 || !ui.footer.table.GetVisible() {
		t.tagfind = nil
		t.tagfindCurrent = nil
		return
	}

	if !ui.footer.table.GetVisible() {
		return
	}

	flags := "ms"
	if !ui.footer.caseBtn.GetActive() {
		flags += "i"
	}

	if !ui.footer.regBtn.GetActive() {
		t.find = regexp.QuoteMeta(t.find)
	}

	if t.Encoding == "binary" {
		t.find = regexp.MustCompile("[ \n\r]+").ReplaceAllString(t.find, "")
		t.find = regexp.MustCompile("(?i)([0-9a-z]{2})").ReplaceAllString(t.find, "$1[ \r\n]*")
	}

	findtext := t.GetText(true)

	//if new text not found prev, reset index
	if findtext != t.findtext {
		t.findindexCurrent = 0
	}
	t.findtext = findtext

	expr := fmt.Sprintf("(?%s)%s", flags, t.find)
	reg, err := regexp.Compile(expr)
	if err != nil {
		log.Println("invalid search query,", err)
		return
	}

	t.findindex = reg.FindAllStringIndex(t.findtext, conf.Search.MaxItems)

	t.tagfind = t.sourcebuffer.CreateTag("find", map[string]interface{}{"background": "#999999"})
	t.tagfindCurrent = t.sourcebuffer.CreateTag("findCurr", map[string]interface{}{"background": "#eeaa00"})

	for i, index := range t.findindex {
		data := []byte(t.findtext)
		if t.Encoding != "binary" {
			index[0] = utf8.RuneCount(data[:index[0]])
			index[1] = utf8.RuneCount(data[:index[1]])
			t.findindex[i] = index
		}
		if i == 0 {
			t.Highlight(i, true)
		} else {
			t.Highlight(i, false)
		}
	}
}

func (t *Tab) onMoveCursor() {
	mark := t.sourcebuffer.GetInsert()
	t.sourcebuffer.GetIterAtMark(&t.cursorPos, mark)
	t.findoffset = t.cursorPos.GetOffset()
	t.findwrap = false

	t.Highlight(t.findindexCurrent, false)
	t.findindexCurrent = -1
}

func (t *Tab) FindNext(next bool) {
	if len(t.findindex) < 2 {
		return
	}

	if t.findindexCurrent > len(t.findindex) {
		t.findindexCurrent = len(t.findindex) - 1
	}

	t.Highlight(t.findindexCurrent, false)

	if next {
		t.findindexCurrent++
		if t.findindexCurrent >= len(t.findindex) {
			t.findindexCurrent = 0
			t.findwrap = true
		}

	} else {
		t.findindexCurrent--
		if t.findindexCurrent < 0 {
			t.findindexCurrent = len(t.findindex) - 1
		}
	}

	index := t.findindex[t.findindexCurrent]
	if !t.findwrap {
		for index[1] < t.findoffset {
			t.findindexCurrent++
			if t.findindexCurrent >= len(t.findindex) {
				t.findindexCurrent = 0
				t.findwrap = true
				break
			}
			index = t.findindex[t.findindexCurrent]
		}
	}

	t.Highlight(t.findindexCurrent, true)
}

func (t *Tab) Highlight(i int, current bool) {
	if i >= len(t.findindex) || i < 0 {
		return
	}
	index := t.findindex[i]
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
