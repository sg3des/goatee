package main

import (
	"fmt"
	"path"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

var (
	menubar *gtk.Widget

	footer     *gtk.Table
	footerFind *gtk.HBox
	footerRepl *gtk.HBox

	findbar *gtk.Entry
	replbar *gtk.Entry
	btnReg  *gtk.ToggleButton
)

func CreateMenu(w *gtk.Window, vbox *gtk.VBox) {
	action_group := gtk.NewActionGroup("my_group")
	ui_manager := CreateUIManager()
	accel_group := ui_manager.GetAccelGroup()
	w.AddAccelGroup(accel_group)
	AddFileMenuActions(action_group)
	AddEditMenuActions(action_group)
	AddChoicesMenuActions(action_group)

	ui_manager.InsertActionGroup(action_group, 0)
	menubar = ui_manager.GetWidget("/MenuBar")

	vbox.PackStart(menubar, false, false, 0)

	vbox.PackEnd(CreateFooter(), false, false, 0)
}

func CreateFooter() *gtk.Table {
	footer = gtk.NewTable(2, 5, false)

	// findbar
	labelReg := gtk.NewLabel("Re")
	labelReg.ModifyFG(gtk.STATE_ACTIVE, gdk.NewColor("red"))
	btnReg = gtk.NewToggleButton()
	btnReg.Add(labelReg)
	btnReg.Connect("toggled", OnFindInput)

	findbar = gtk.NewEntryWithBuffer(gtk.NewEntryBuffer(""))
	findbar.Connect("changed", OnFindInput)

	btnNext := gtk.NewButton()
	btnNext.SetSizeRequest(20, 20)
	btnNext.Add(gtk.NewArrow(gtk.ARROW_DOWN, gtk.SHADOW_NONE))
	btnNext.Clicked(OnFindNext)

	btnPrev := gtk.NewButton()
	btnPrev.SetSizeRequest(20, 20)
	btnPrev.Add(gtk.NewArrow(gtk.ARROW_UP, gtk.SHADOW_NONE))
	btnPrev.Clicked(OnFindPrev)

	btnClose := gtk.NewButton()
	btnClose.SetSizeRequest(20, 20)
	btnClose.Add(gtk.NewImageFromStock(gtk.STOCK_CLOSE, gtk.ICON_SIZE_BUTTON))
	btnClose.Clicked(OnMenuFind)

	// replacebar
	replbar = gtk.NewEntryWithBuffer(gtk.NewEntryBuffer(""))
	replbar.Connect("changed", OnFindInput)

	btnRepl := gtk.NewButton()
	btnRepl.SetSizeRequest(20, 20)
	btnRepl.Add(gtk.NewImageFromIconName("text-changelog", gtk.ICON_SIZE_BUTTON))

	btnReplAll := gtk.NewButton()
	btnReplAll.SetSizeRequest(20, 20)
	btnReplAll.Add(gtk.NewImageFromIconName("text-plain", gtk.ICON_SIZE_BUTTON))
	// btnRepl.Clicked(OnMenuFind)

	// pack to table
	footer.Attach(btnReg, 0, 1, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
	footer.Attach(findbar, 1, 2, 0, 1, gtk.EXPAND|gtk.FILL, gtk.FILL, 0, 0)
	footer.Attach(btnNext, 2, 3, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
	footer.Attach(btnPrev, 3, 4, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
	footer.Attach(btnClose, 4, 5, 0, 1, gtk.FILL, gtk.FILL, 0, 0)

	footer.Attach(replbar, 1, 2, 1, 2, gtk.EXPAND|gtk.FILL, gtk.FILL, 0, 0)
	footer.Attach(btnRepl, 2, 3, 1, 2, gtk.FILL, gtk.FILL, 0, 0)
	footer.Attach(btnReplAll, 3, 4, 1, 2, gtk.FILL, gtk.FILL, 0, 0)

	return footer
}

func OnFindInput() {
	currentTab().Find(findbar.GetText())
}

func OnFindNext() {
	currentTab().FindNext(true)
}

func OnFindPrev() {
	currentTab().FindNext(false)
}

func CreateUIManager() *gtk.UIManager {
	UI_INFO := `
<ui>
  <menubar name='MenuBar'>
    <menu action='FileMenu'>
      <menuitem action='NewTab' />
      <menuitem action='CloseTab' />
      <menuitem action='FileOpen' />
      <menuitem action='FileSave' />
      <menuitem action='FileSaveAs' />
      <separator />
      <menuitem action='FileQuit' />
    </menu>
    <menu action='EditMenu'>
      <menuitem action='Find'/>
      <menuitem action='FindNext'/>
      <menuitem action='FindPrev'/>
      <separator />
      <menuitem action='Replace'/>
    </menu>
    <menu action='ChoicesMenu'>
      <menuitem action='ChoiceOne'/>
      <menuitem action='ChoiceTwo'/>
      <menuitem action='ChoiceThree'/>
      <separator />
      <menuitem action='ChoiceToggle'/>
    </menu>
  </menubar>
</ui>
`
	ui_manager := gtk.NewUIManager()
	ui_manager.AddUIFromString(UI_INFO)
	return ui_manager
}

func OnMenuFileQuit() {
	exit()
}

func OnMenuFileOpen() {
	dialog := gtk.NewFileChooserDialog("open", window, gtk.FILE_CHOOSER_ACTION_OPEN, "open file", gtk.RESPONSE_OK)

	dialog.Run()

	filename := dialog.GetFilename()

	dialog.Destroy()

	if len(filename) > 0 {
		NewTab(filename)
	}
}

func OnMenuFileSave() {
	// n := notebook.GetCurrentPage()
	t := currentTab()
	if t == nil {
		return
	}
	if t.File == nil {
		filename := dialogSave()
		if len(filename) == 0 {
			return
		}

		t.Filename = filename
		t.label.SetText(path.Base(filename))
		t.label.SetTooltipText(filename)
	}
	t.Save()
}

func OnMenuFileSaveAs() {
	t := currentTab()

	filename := dialogSave()
	if len(filename) == 0 {
		return
	}

	t.Filename = filename
	t.label.SetText(path.Base(filename))
	t.label.SetTooltipText(filename)
}

func dialogSave() string {
	dialog := gtk.NewFileChooserDialog("save", window, gtk.FILE_CHOOSER_ACTION_SAVE, "save file", gtk.RESPONSE_OK)
	dialog.Run()
	filename := dialog.GetFilename()
	dialog.Destroy()

	return filename
}

func OnMenuNewTab() {
	NewTab("")
	fmt.Println(len(tabs))
}

func OnMenuCloseTab() {
	closeCurrentTab()

	if len(tabs) == 0 {
		exit()
	}
}

func OnMenuFind() {
	footer.SetVisible(true)
	// footerRepl.SetVisible(true)
	findbar.GrabFocus()

	// if footer.GetVisible() {
	// 	footer.SetVisible(false)
	// 	currentTab().sourceview.GrabFocus()
	// } else {
	// 	footer.SetVisible(true)
	// 	findbar.GrabFocus()
	// }
}

func AddFileMenuActions(action_group *gtk.ActionGroup) {
	action_group.AddAction(gtk.NewAction("FileMenu", "File", "", ""))

	action_newtab := gtk.NewAction("NewTab", "New Tab", "", "")
	action_newtab.Connect("activate", OnMenuNewTab)
	action_group.AddActionWithAccel(action_newtab, "<control>t")

	action_closetab := gtk.NewAction("CloseTab", "Close Tab", "", "")
	action_closetab.Connect("activate", OnMenuCloseTab)
	action_group.AddActionWithAccel(action_closetab, "<control>w")

	action_fileopen := gtk.NewAction("FileOpen", "", "", gtk.STOCK_OPEN)
	action_fileopen.Connect("activate", OnMenuFileOpen)
	action_group.AddActionWithAccel(action_fileopen, "")

	action_filesave := gtk.NewAction("FileSave", "", "", gtk.STOCK_SAVE)
	action_filesave.Connect("activate", OnMenuFileSave)
	action_group.AddActionWithAccel(action_filesave, "")

	action_filesaveas := gtk.NewAction("FileSaveAs", "", "", gtk.STOCK_SAVE_AS)
	action_filesaveas.Connect("activate", OnMenuFileSaveAs)
	action_group.AddActionWithAccel(action_filesaveas, "")

	action_filequit := gtk.NewAction("FileQuit", "", "", gtk.STOCK_QUIT)
	action_filequit.Connect("activate", OnMenuFileQuit)
	action_group.AddActionWithAccel(action_filequit, "")
}

func AddEditMenuActions(action_group *gtk.ActionGroup) {
	action_group.AddAction(gtk.NewAction("EditMenu", "Edit", "", ""))

	action_find := gtk.NewAction("Find", "Find...", "", gtk.STOCK_FIND)
	action_find.Connect("activate", OnMenuFind)
	action_group.AddActionWithAccel(action_find, "")

	action_findnext := gtk.NewAction("FindNext", "Find Next", "", "")
	action_findnext.Connect("activate", OnFindNext)
	action_group.AddActionWithAccel(action_findnext, "F3")

	action_findprev := gtk.NewAction("FindPrev", "Find Previus", "", "")
	action_findprev.Connect("activate", OnFindPrev)
	action_group.AddActionWithAccel(action_findprev, "<shift>F3")

	action_repl := gtk.NewAction("Replace", "Replace...", "", gtk.STOCK_FIND_AND_REPLACE)
	action_repl.Connect("activate", OnMenuFind)
	action_group.AddActionWithAccel(action_repl, "<control>h")
}

func AddChoicesMenuActions(action_group *gtk.ActionGroup) {
	action_group.AddAction(gtk.NewAction("ChoicesMenu", "Choices", "", ""))

	var ra_list []*gtk.RadioAction
	ra_one := gtk.NewRadioAction("ChoiceOne", "One", "", "", 1)
	ra_list = append(ra_list, ra_one)

	ra_two := gtk.NewRadioAction("ChoiceTwo", "Two", "", "", 2)
	ra_list = append(ra_list, ra_two)

	ra_three := gtk.NewRadioAction("ChoiceThree", "Three", "", "", 2)
	ra_list = append(ra_list, ra_three)

	var sl *glib.SList
	for _, ra := range ra_list {
		ra.SetGroup(sl)
		sl = ra.GetGroup()
		action_group.AddAction(ra)
	}

	ra_last := gtk.NewToggleAction("ChoiceToggle", "Toggle", "", "")
	ra_last.SetActive(true)
	action_group.AddAction(ra_last)
}
