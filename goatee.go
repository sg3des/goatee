package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
	"unsafe"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
	gsv "github.com/mattn/go-gtk/gtksourceview"

	"github.com/endeveit/enca"
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

	targets = []gtk.TargetEntry{
		{"text/uri-list", 0, 0},
		{"STRING", 0, 1},
		{"text/plain", 0, 2},
	}

	analyzer *enca.EncaAnalyser

	languages = gsv.SourceLanguageManagerGetDefault().GetLanguageIds()

	err error
)

func init() {
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	user, _ := user.Current()

	gvfsPath = fmt.Sprintf(gvfsPath, user.Uid)
}

func main() {
	gtk.Init(nil)
	window = CreateWindow()
	window.Connect("destroy", gtk.MainQuit)
	window.Connect("check-resize", windowResize)

	window.ShowAll()
	menubar.SetVisible(false)
	footer.SetVisible(false)

	analyzer, err = enca.New("ru")
	if err != nil {
		log.Fatalln("failed load chaser analyzer", err)
	}

	NewTab(filename)

	gtk.Main()
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

func windowResize() {
	window.GetSize(&width, &height)
	notebook.SetSizeRequest(width, height)
	homogenousTabs()
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

type Tab struct {
	Filename string
	File     *os.File
	Data     []byte
	Encoding string
	Language string
	ReadOnly bool

	color        *gdk.Color
	swin         *gtk.ScrolledWindow
	label        *gtk.Label
	sourcebuffer *gsv.SourceBuffer
	sourceview   *gsv.SourceView

	tagfind *gtk.TextTag
}

func NewTab(filename string) {
	var newfile bool

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

		t.Encoding, err = analyzer.FromBytes(t.Data, enca.NAME_STYLE_HUMAN)
		analyzer.Free()
		if err != nil {
			t.Encoding = "binary"
			t.Data = []byte(hex.Dump(t.Data))
		} else {
			t.Language = t.DetectLanguage()

			if t.Encoding != "ASCII" && t.Encoding != "binary" {
				t.Data, err = changeEncoding(t.Data, "utf-8", t.Encoding)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
			// t.Text = string(t.Data)
		}
	}

	ct := currentTab()
	if ct != nil && !newfile && len(ct.Data) == 0 {
		closeTab()
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
	t.sourceview.SetHighlightCurrentLine(true)
	t.sourceview.ModifyFontEasy("Liberation Mono 8")
	if t.Encoding != "binary" {
		t.sourceview.SetShowLineNumbers(true)
		t.sourceview.SetWrapMode(gtk.WRAP_WORD_CHAR)
		t.sourceview.SetTabWidth(2)
		t.sourceview.SetInsertSpacesInsteadOfTabs(false)
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

	// t.label.RenderIcon("object-locked", gtk.GTK_ICON_SIZE_SMALL_TOOLBAR, "lock")
	// if t.ReadOnly {
	// 	t.labelColorReadOnly()
	// }
	// t.label.SetUSize(10000, 10)

	if newfile {
		t.updateTabColor(255, 200, 0)
	} else {
		t.resetTabColor()
	}

	n := notebook.AppendPage(t.swin, t.label)
	notebook.ShowAll()
	notebook.SetCurrentPage(n)
	t.sourceview.GrabFocus()

	tabs = append(tabs, t)

	// homogenousTabs()
}

func toHex(data []byte) (hexdata string) {
	reader := bytes.NewReader(data)
	for err == nil {
		var b byte
		b, err = reader.ReadByte()
		// fmt.Printf("%02x ", b)
		hexdata += fmt.Sprintf("%02x ", b)
		// fmt.Println(hexdata)
	}
	// fmt.Println(string(hexdata))
	return
}

func (t *Tab) DetectLanguage() string {
	if len(languages) == 0 {
		return ""
	}

	ext := path.Ext(t.Filename)
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

func (t *Tab) DragAndDrop() {
	t.sourceview.DragDestSet(gtk.DEST_DEFAULT_MOTION|gtk.DEST_DEFAULT_HIGHLIGHT|gtk.DEST_DEFAULT_DROP, targets, gdk.ACTION_COPY)
	t.sourceview.DragDestAddUriTargets()
	t.sourceview.Connect("drag-data-received", func(ctx *glib.CallbackContext) {
		sdata := gtk.NewSelectionDataFromNative(unsafe.Pointer(ctx.Args(3)))
		if sdata != nil {
			a := (*[2000]uint8)(sdata.GetData())
			files := strings.Split(string(a[:sdata.GetLength()-1]), "\n")
			for i := range files {
				filename, _, err := glib.FilenameFromUri(files[i])
				if err == nil && len(filename) > 0 {
					// filename = strings.TrimSpace(filename)
					if len(filename) > 0 {
						NewTab(filename)
					}
				} else {
					filename = strings.TrimSpace(files[i])

					u, err := url.Parse(filename)
					if err != nil {
						fmt.Println("failed parse path to file", err)
						continue
					}
					filename = path.Join(gvfsPath, fmt.Sprintf("%s:host=%s", u.Scheme, u.Host), u.Path)
					fmt.Println(filename)
					NewTab(filename)
					// filename = path.Join(gvfsPath, files[i])
					// NewTab(filename)
					// fmt.Println()
				}
			}
		}
	})
}

func (t *Tab) onchange() {
	t.Data = t.GetText()
	t.updateTabColor(255, 0, 0)
}

func (t *Tab) updateTabColor(r, g, b uint16) {
	if t.ReadOnly {
		r -= 50
		g += 100
		b += 100
		// t.label.
	}
	r = uint16(math.Pow(float64(r), 2))
	g = uint16(math.Pow(float64(g), 2))
	b = uint16(math.Pow(float64(b), 2))

	// fmt.Println(r, float64(r), math.Pow(float64(r), 2))

	color := gdk.NewColorRGB(r, g, b)
	t.label.ModifyFG(gtk.STATE_NORMAL, color)
	// t.label.ModifyFG(gtk.STATE_INSENSITIVE, color)
	t.label.ModifyFG(gtk.STATE_PRELIGHT, color)
	t.label.ModifyFG(gtk.STATE_SELECTED, color)
	t.label.ModifyFG(gtk.STATE_ACTIVE, color)
	// t.label.ModifyBG(state, color)
}

// func (t *Tab) updateTabBGColor(r, g, b uint8) {
// 	color := gdk.NewColorRGB(r, g, b)
// }

// func (t *Tab) labelColorReadOnly() {

// 	color := gdk.NewColorRGB(0, 0, 255)
// 	fmt.Println(color)
// 	t.label.ModifyBG(gtk.STATE_NORMAL, color)
// 	t.label.ModifyBG(gtk.STATE_PRELIGHT, color)
// 	t.label.ModifyBG(gtk.STATE_SELECTED, color)
// 	t.label.ModifyBG(gtk.STATE_ACTIVE, color)
// 	t.label.ModifyBG(gtk.STATE_INSENSITIVE, color)
// 	// t.label.ModifyBase(state, color)
// 	t.label.ModifyBG(gtk.STATE_SELECTED, color)
// }

func (t *Tab) resetTabColor() {
	var color *gdk.Color
	if t.ReadOnly {
		color = gdk.NewColorRGB(30400, 30400, 30400)
	} else {
		color = gdk.NewColorRGB(40000, 40000, 40000)
	}
	t.label.ModifyFG(gtk.STATE_NORMAL, color)
	// t.label.ModifyFG(gtk.STATE_INSENSITIVE, color)
	t.label.ModifyFG(gtk.STATE_PRELIGHT, color)
	t.label.ModifyFG(gtk.STATE_SELECTED, color)
	t.label.ModifyFG(gtk.STATE_ACTIVE, color)
}

func (t *Tab) Save() {
	var text []byte
	if t.Encoding == "binary" {
		log.Println("sorry, binary data save not yet done")
		return
	} else if t.ReadOnly {
		log.Println("sorry, file is open readonly mode")
		return
	} else {
		text, err = changeEncoding(t.GetText(), t.Encoding, "utf-8")
		if err != nil {
			log.Println("failed restore encoding, save failed", err)
			return
		}
	}

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
	t.resetTabColor()
}

func (t *Tab) GetText() []byte {
	var start gtk.TextIter
	var end gtk.TextIter

	t.sourcebuffer.GetStartIter(&start)
	t.sourcebuffer.GetEndIter(&end)
	return []byte(t.sourcebuffer.GetText(&start, &end, true))
}

func (t *Tab) Find(substr string) {
	// text := string(t.GetText())

	if t.tagfind != nil {
		tabletag := t.sourcebuffer.GetTagTable()
		tabletag.Remove(t.tagfind)
	}

	if len(substr) == 0 {
		t.tagfind = nil
		return
	}

	reg, err := regexp.Compile(substr)
	if err != nil {
		t.tagfind = nil
		log.Println("invalid regexp query", err)
		return
	}

	t.tagfind = t.sourcebuffer.CreateTag("find", map[string]string{"background": "#999999"})
	matches := reg.FindAllIndex(t.GetText(), -1)

	for n, index := range matches {
		var start gtk.TextIter
		var end gtk.TextIter
		t.sourcebuffer.GetIterAtOffset(&start, index[0])
		t.sourcebuffer.GetIterAtOffset(&end, index[1])
		t.sourcebuffer.ApplyTag(t.tagfind, &start, &end)
		if n == 0 {
			t.sourceview.ScrollToIter(&start, 0, false, 0, 0)
		}
	}
	fmt.Println(matches)

	// var i int
	// var offset int
	// for {
	// 	i++
	// 	n := strings.Index(text, substr)
	// 	if n == -1 {
	// 		t.tagfind = nil
	// 		break
	// 	}

	// 	var start gtk.TextIter
	// 	var end gtk.TextIter
	// 	t.sourcebuffer.GetIterAtOffset(&start, offset+n)
	// 	t.sourcebuffer.GetIterAtOffset(&end, offset+n+len(substr))

	// 	t.sourcebuffer.ApplyTag(t.tagfind, &start, &end)
	// 	start.Free()
	// 	end.Free()

	// 	offset += n + len(substr)
	// 	text = text[offset:]

	// 	if i == 1 {
	// 		t.sourceview.ScrollToIter(&start, 0, false, 0, 0)
	// 	}
	// }
}

func closeTab() {
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
	if len(tabs) == 0 {
		return
	}

	tabwidth := (width - len(tabs)*5) / len(tabs)
	for _, t := range tabs {
		t.label.SetSizeRequest(tabwidth-1, 12)
	}

}
