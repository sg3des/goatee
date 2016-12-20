package main

import (
	"fmt"
	"path"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
)

type UI struct {
	window     *gtk.Window
	accelGroup *gtk.AccelGroup

	vbox    *gtk.VBox
	menubar *gtk.Widget

	notebook *gtk.Notebook

	footer struct {
		table *gtk.Table

		findEntry *gtk.Entry
		replEntry *gtk.Entry

		regBtn  *gtk.ToggleButton
		caseBtn *gtk.ToggleButton

		findNextBtn *gtk.Button
		findPrevBtn *gtk.Button

		replBtn    *gtk.Button
		replAllBtn *gtk.Button

		closeBtn *gtk.Button
	}
}

func CreateUI() *UI {
	ui := &UI{}
	ui.window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	ui.window.SetDefaultSize(700, 300)

	ui.vbox = gtk.NewVBox(false, 0)
	ui.notebook = gtk.NewNotebook()
	ui.vbox.Add(ui.notebook)
	ui.window.Add(ui.vbox)

	ui.vbox.PackStart(ui.createMenubar(), false, false, 0)

	ui.vbox.PackEnd(ui.createFooter(), false, false, 0)

	ui.window.Connect("destroy", exit)
	ui.window.Connect("check-resize", ui.windowResize)

	ui.window.ShowAll()
	ui.menubar.SetVisible(false)
	ui.footer.table.SetVisible(false)
	return ui
}

func (ui *UI) createMenubar() *gtk.Widget {
	actionGroup := gtk.NewActionGroup("my_group")
	uiManager := ui.createUIManager()

	ui.accelGroup = uiManager.GetAccelGroup()
	ui.window.AddAccelGroup(ui.accelGroup)
	ui.addFileMenuActions(actionGroup)
	ui.addEditMenuActions(actionGroup)

	uiManager.InsertActionGroup(actionGroup, 0)
	ui.menubar = uiManager.GetWidget("/MenuBar")
	return ui.menubar
}

func (ui *UI) createUIManager() *gtk.UIManager {
	UIxml := `
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
      <menuitem action='ReplaceOne'/>
      <menuitem action='ReplaceAll'/>
    </menu>
  </menubar>
</ui>
`
	uiManager := gtk.NewUIManager()
	uiManager.AddUIFromString(UIxml)
	return uiManager
}

func (ui *UI) addFileMenuActions(actionGroup *gtk.ActionGroup) {
	actionGroup.AddAction(gtk.NewAction("FileMenu", "File", "", ""))

	actionNewtab := gtk.NewAction("NewTab", "New Tab", "", "")
	actionNewtab.Connect("activate", OnMenuNewTab)
	actionGroup.AddActionWithAccel(actionNewtab, "<control>t")

	actionClosetab := gtk.NewAction("CloseTab", "Close Tab", "", "")
	actionClosetab.Connect("activate", OnMenuCloseTab)
	actionGroup.AddActionWithAccel(actionClosetab, "<control>w")

	actionFileopen := gtk.NewAction("FileOpen", "", "", gtk.STOCK_OPEN)
	actionFileopen.Connect("activate", OnMenuFileOpen)
	actionGroup.AddActionWithAccel(actionFileopen, "")

	actionFilesave := gtk.NewAction("FileSave", "", "", gtk.STOCK_SAVE)
	actionFilesave.Connect("activate", OnMenuFileSave)
	actionGroup.AddActionWithAccel(actionFilesave, "")

	actionFilesaveas := gtk.NewAction("FileSaveAs", "", "", gtk.STOCK_SAVE_AS)
	actionFilesaveas.Connect("activate", OnMenuFileSaveAs)
	actionGroup.AddActionWithAccel(actionFilesaveas, "")

	actionFilequit := gtk.NewAction("FileQuit", "", "", gtk.STOCK_QUIT)
	actionFilequit.Connect("activate", OnMenuFileQuit)
	actionGroup.AddActionWithAccel(actionFilequit, "")
}

func (ui *UI) addEditMenuActions(actionGroup *gtk.ActionGroup) {
	actionGroup.AddAction(gtk.NewAction("EditMenu", "Edit", "", ""))

	actionFind := gtk.NewAction("Find", "Find...", "", gtk.STOCK_FIND)
	actionFind.Connect("activate", OnMenuFind)
	actionGroup.AddActionWithAccel(actionFind, "")

	actionFindnext := gtk.NewAction("FindNext", "Find Next", "", "")
	actionFindnext.Connect("activate", OnFindNext)
	actionGroup.AddActionWithAccel(actionFindnext, "F3")

	actionFindprev := gtk.NewAction("FindPrev", "Find Previus", "", "")
	actionFindprev.Connect("activate", OnFindPrev)
	actionGroup.AddActionWithAccel(actionFindprev, "<shift>F3")

	actionRepl := gtk.NewAction("Replace", "Replace...", "", gtk.STOCK_FIND_AND_REPLACE)
	actionRepl.Connect("activate", OnMenuReplace)
	actionGroup.AddActionWithAccel(actionRepl, "<control>h")

	actionReplOne := gtk.NewAction("ReplaceOne", "Replace One", "", "")
	actionReplOne.Connect("activate", OnReplaceOne)
	actionGroup.AddActionWithAccel(actionReplOne, "<control><shift>H")

	actionReplAll := gtk.NewAction("ReplaceAll", "Replace All", "", "")
	actionReplAll.Connect("activate", OnReplaceAll)
	actionGroup.AddActionWithAccel(actionReplAll, "<control><alt>Return")
}

func exit() {
	for _, t := range tabs {
		t.File.Close()
	}

	gtk.MainQuit()
}

func (ui *UI) windowResize() {
	ui.window.GetSize(&width, &height)
	ui.notebook.SetSizeRequest(width, height)
	ui.homogenousTabs()
}

func (ui *UI) homogenousTabs() {
	if len(tabs) == 0 || !conf.Tabs.Homogenous {
		return
	}

	tabwidth := (width - len(tabs)*6) / len(tabs)
	leftwidth := (width - len(tabs)*6) % len(tabs)

	for _, t := range tabs {
		if leftwidth > 0 {
			t.label.SetSizeRequest(tabwidth+1, 12)
			leftwidth--
		} else {
			t.label.SetSizeRequest(tabwidth, 12)
		}
	}

}

func (ui *UI) createFooter() *gtk.Table {
	ui.footer.table = gtk.NewTable(2, 6, false)

	// findbar
	labelReg := gtk.NewLabel("Re")
	labelReg.ModifyFG(gtk.STATE_ACTIVE, gdk.NewColor("red"))
	ui.footer.regBtn = gtk.NewToggleButton()
	ui.footer.regBtn.Add(labelReg)
	ui.footer.regBtn.Connect("toggled", OnFindInput)

	labelCase := gtk.NewLabel("A")
	labelCase.ModifyFG(gtk.STATE_ACTIVE, gdk.NewColor("red"))
	ui.footer.caseBtn = gtk.NewToggleButton()
	ui.footer.caseBtn.Add(labelCase)
	ui.footer.caseBtn.SetSizeRequest(20, 20)
	ui.footer.caseBtn.Connect("toggled", OnFindInput)

	ui.footer.findEntry = gtk.NewEntryWithBuffer(gtk.NewEntryBuffer(""))
	ui.footer.findEntry.Connect("changed", OnFindInput)

	ui.footer.findNextBtn = gtk.NewButton()
	ui.footer.findNextBtn.SetSizeRequest(20, 20)
	ui.footer.findNextBtn.Add(gtk.NewArrow(gtk.ARROW_DOWN, gtk.SHADOW_NONE))
	ui.footer.findNextBtn.Clicked(OnFindNext)

	ui.footer.findPrevBtn = gtk.NewButton()
	ui.footer.findPrevBtn.SetSizeRequest(20, 20)
	ui.footer.findPrevBtn.Add(gtk.NewArrow(gtk.ARROW_UP, gtk.SHADOW_NONE))
	ui.footer.findPrevBtn.Clicked(OnFindPrev)

	ui.footer.closeBtn = gtk.NewButton()
	ui.footer.closeBtn.SetSizeRequest(20, 20)
	ui.footer.closeBtn.Add(gtk.NewImageFromStock(gtk.STOCK_CLOSE, gtk.ICON_SIZE_BUTTON))
	ui.footer.closeBtn.Clicked(ui.footerClose)
	ui.footer.closeBtn.AddAccelerator("activate", ui.accelGroup, gdk.KEY_Escape, 0, gtk.ACCEL_VISIBLE)

	// replacebar
	ui.footer.replEntry = gtk.NewEntryWithBuffer(gtk.NewEntryBuffer(""))
	// ui.footer.replEntry.Connect("changed", OnFindInput)

	ui.footer.replBtn = gtk.NewButton()
	ui.footer.replBtn.SetSizeRequest(20, 20)
	ui.footer.replBtn.Add(gtk.NewImageFromIconName("text-changelog", gtk.ICON_SIZE_BUTTON))
	ui.footer.replBtn.Clicked(OnReplaceOne)

	ui.footer.replAllBtn = gtk.NewButton()
	ui.footer.replAllBtn.SetSizeRequest(20, 20)
	ui.footer.replAllBtn.Add(gtk.NewImageFromIconName("text-plain", gtk.ICON_SIZE_BUTTON))
	ui.footer.replAllBtn.Clicked(OnReplaceAll)
	// btnRepl.Clicked(OnMenuFind)

	// pack to table
	ui.footer.table.Attach(ui.footer.regBtn, 0, 1, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
	ui.footer.table.Attach(ui.footer.caseBtn, 1, 2, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
	ui.footer.table.Attach(ui.footer.findEntry, 2, 3, 0, 1, gtk.EXPAND|gtk.FILL, gtk.FILL, 0, 0)
	ui.footer.table.Attach(ui.footer.findNextBtn, 3, 4, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
	ui.footer.table.Attach(ui.footer.findPrevBtn, 4, 5, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
	ui.footer.table.Attach(ui.footer.closeBtn, 5, 6, 0, 1, gtk.FILL, gtk.FILL, 0, 0)

	ui.footer.table.Attach(ui.footer.replEntry, 2, 3, 1, 2, gtk.EXPAND|gtk.FILL, gtk.FILL, 0, 0)
	ui.footer.table.Attach(ui.footer.replBtn, 3, 4, 1, 2, gtk.FILL, gtk.FILL, 0, 0)
	ui.footer.table.Attach(ui.footer.replAllBtn, 4, 5, 1, 2, gtk.FILL, gtk.FILL, 0, 0)

	return ui.footer.table
}

func OnMenuFind() {
	ui.footer.table.SetVisible(true)
	ui.footer.replEntry.SetVisible(false)
	ui.footer.replBtn.SetVisible(false)
	ui.footer.replAllBtn.SetVisible(false)

	ui.footer.findEntry.GrabFocus()
}

func OnMenuReplace() {
	ui.footer.table.SetVisible(true)
	ui.footer.replEntry.SetVisible(true)
	ui.footer.replBtn.SetVisible(true)
	ui.footer.replAllBtn.SetVisible(true)

	ui.footer.replEntry.GrabFocus()
}

func (ui *UI) footerClose() {
	ui.footer.table.SetVisible(false)
}

func OnFindInput() {
	currentTab().Find(ui.footer.findEntry.GetText())
}

func OnFindNext() {
	currentTab().FindNext(true)
}

func OnFindPrev() {
	currentTab().FindNext(false)
}

func OnReplaceOne() {
	currentTab().Replace(false)
}

func OnReplaceAll() {
	currentTab().Replace(true)
}

func OnMenuFileOpen() {
	dialog := gtk.NewFileChooserDialog("open", ui.window, gtk.FILE_CHOOSER_ACTION_OPEN, "open file", gtk.RESPONSE_OK)

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
	dialog := gtk.NewFileChooserDialog("save", ui.window, gtk.FILE_CHOOSER_ACTION_SAVE, "save file", gtk.RESPONSE_OK)

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

func OnMenuFileQuit() {
	exit()
}
