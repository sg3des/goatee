package main

import (
	"log"
	"path"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
)

type UI struct {
	window      *gtk.Window
	accelGroup  *gtk.AccelGroup
	actionGroup *gtk.ActionGroup

	vbox     *gtk.VBox
	menubar  *gtk.Widget
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
	ui.window.SetDefaultSize(600, 300)

	ui.vbox = gtk.NewVBox(false, 0)
	ui.vbox.PackStart(ui.createUIManager(), false, false, 0)
	ui.notebook = gtk.NewNotebook()
	ui.vbox.PackStart(ui.notebook, true, true, 0)
	ui.vbox.PackStart(ui.createFooter(), false, false, 0)
	ui.window.Add(ui.vbox)

	ui.window.Connect("destroy", ui.Quit)
	ui.window.Connect("check-resize", ui.windowResize)

	ui.window.ShowAll()

	ui.footer.table.SetVisible(false)
	ui.menubar.SetVisible(conf.UI.MenuBarVisible)
	return ui
}

func (ui *UI) NewTab() {
	NewTab("")
}
func (ui *UI) Open() {
	dialog := gtk.NewFileChooserDialog("open", ui.window, gtk.FILE_CHOOSER_ACTION_OPEN, "open file", gtk.RESPONSE_OK)

	dialog.Run()
	filename := dialog.GetFilename()
	dialog.Destroy()

	if len(filename) > 0 {
		NewTab(filename)
	}
}
func (ui *UI) Save() {
	t := currentTab()

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
func (ui *UI) SaveAs() {
	t := currentTab()

	filename := dialogSave()
	if len(filename) == 0 {
		return
	}

	t.Filename = filename
	t.label.SetText(path.Base(filename))
	t.label.SetTooltipText(filename)
	t.Save()
}
func (ui *UI) CloseTab() {
	closeCurrentTab()

	if len(tabs) == 0 {
		gtk.MainQuit()
	}
}
func (ui *UI) Quit() {
	for _, t := range tabs {
		t.File.Close()
	}
	gtk.MainQuit()
}

func (ui *UI) Find() {
	currentTab().Find()
}
func (ui *UI) FindNext() {
	currentTab().FindNext(true)
}
func (ui *UI) FindPrev() {
	currentTab().FindNext(false)
}
func (ui *UI) ReplaceOne() {
	currentTab().Replace(false)
}
func (ui *UI) ReplaceAll() {
	currentTab().Replace(true)
}

func (ui *UI) ToggleMenuBar() {
	conf.UI.MenuBarVisible = !conf.UI.MenuBarVisible
	ui.menubar.SetVisible(conf.UI.MenuBarVisible)

}
func (ui *UI) ToggleStatusBar() {
	log.Println("statusbar")
	// conf.UI.StatusBarVisible = !conf.UI.StatusBarVisible
	// ui.statusbar.SetVisible(conf.UI.StatusBarVisible)
	// ui.menu.statusbar.SetActive(conf.UI.StatusBarVisible)
}

func dialogSave() string {
	dialog := gtk.NewFileChooserDialog("save", ui.window, gtk.FILE_CHOOSER_ACTION_SAVE, "save file", gtk.RESPONSE_OK)

	dialog.Run()
	filename := dialog.GetFilename()
	dialog.Destroy()

	return filename
}

func (ui *UI) createUIManager() *gtk.Widget {

	UIxml := `
<ui>
	<menubar name='MenuBar'>

		<menu action='File'>
			<menuitem action='NewTab' />
			<menuitem action='Open' />
			<menuitem action='Save' />
			<menuitem action='SaveAs' />
			<separator />
			<menuitem action='CloseTab' />
			<menuitem action='Quit' />
		</menu>

		<menu action='Edit'>
			<menuitem action='Find'/>
			<menuitem action='FindNext'/>
			<menuitem action='FindPrev'/>
			<separator />
			<menuitem action='Replace'/>
			<menuitem action='ReplaceOne'/>
			<menuitem action='ReplaceAll'/>
		</menu>

		<menu name='View' action='View'>
			<menuitem action='Menubar'/>
		</menu>

	</menubar>
</ui>
`
	uiManager := gtk.NewUIManager()
	uiManager.AddUIFromString(UIxml)

	ui.accelGroup = uiManager.GetAccelGroup()
	ui.window.AddAccelGroup(ui.accelGroup)

	ui.actionGroup = gtk.NewActionGroup("my_group")
	uiManager.InsertActionGroup(ui.actionGroup, 0)

	// File
	ui.actionGroup.AddAction(gtk.NewAction("File", "File", "", ""))

	ui.newAction("NewTab", "New Tab", "<control>t", ui.NewTab)
	ui.newActionStock("Open", gtk.STOCK_OPEN, "", ui.Open)
	ui.newAction("Save", gtk.STOCK_SAVE, "", ui.Save)
	ui.newAction("SaveAs", gtk.STOCK_SAVE_AS, "", ui.SaveAs)
	ui.newAction("CloseTab", "Close Tab", "<control>w", ui.CloseTab)
	ui.newActionStock("Quit", gtk.STOCK_QUIT, "", ui.Quit)

	// Edit
	ui.actionGroup.AddAction(gtk.NewAction("Edit", "Edit", "", ""))

	ui.newActionStock("Find", gtk.STOCK_FIND, "", ui.ShowFindbar)
	ui.newAction("FindNext", "Find Next", "F3", ui.FindNext)
	ui.newAction("FindPrev", "Find Previous", "<shift>F3", ui.FindPrev)

	ui.newActionStock("Replace", gtk.STOCK_FIND_AND_REPLACE, "<control>h", ui.ShowReplbar)
	ui.newAction("ReplaceOne", "Replace One", "<control><shift>h", ui.ReplaceOne)
	ui.newAction("ReplaceAll", "Replace All", "<control><alt>Return", ui.ReplaceAll)

	// View
	ui.actionGroup.AddAction(gtk.NewAction("View", "View", "", ""))
	// ui.actionGroup.AddAction(gtk.NewAction("Encoding", "Encoding", "", ""))

	ui.newToggleAction("Menubar", "Menubar", "<control>M", conf.UI.MenuBarVisible, ui.ToggleMenuBar)

	ui.menubar = uiManager.GetWidget("/MenuBar")

	return ui.menubar
}

func (ui *UI) newAction(dst, label, accel string, f func()) {
	action := gtk.NewAction(dst, label, "", "")
	action.Connect("activate", f)
	ui.actionGroup.AddActionWithAccel(action, accel)
}

func (ui *UI) newActionStock(dst, stock, accel string, f func()) {
	action := gtk.NewAction(dst, "", "", stock)
	action.Connect("activate", f)
	ui.actionGroup.AddActionWithAccel(action, accel)
}

func (ui *UI) newToggleAction(dst, label, accel string, state bool, f func()) {
	action := gtk.NewToggleAction(dst, label, "", "")
	action.SetActive(state)
	action.Connect("activate", f)
	ui.actionGroup.AddActionWithAccel(&action.Action, accel)
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
	ui.footer.regBtn.Connect("toggled", ui.Find)

	labelCase := gtk.NewLabel("A")
	labelCase.ModifyFG(gtk.STATE_ACTIVE, gdk.NewColor("red"))
	ui.footer.caseBtn = gtk.NewToggleButton()
	ui.footer.caseBtn.Add(labelCase)
	ui.footer.caseBtn.SetSizeRequest(20, 20)
	ui.footer.caseBtn.Connect("toggled", ui.Find)

	ui.footer.findEntry = gtk.NewEntryWithBuffer(gtk.NewEntryBuffer(""))
	ui.footer.findEntry.Connect("changed", ui.Find)

	ui.footer.findNextBtn = gtk.NewButton()
	ui.footer.findNextBtn.SetSizeRequest(20, 20)
	ui.footer.findNextBtn.Add(gtk.NewArrow(gtk.ARROW_DOWN, gtk.SHADOW_NONE))
	ui.footer.findNextBtn.Clicked(ui.FindNext)

	ui.footer.findPrevBtn = gtk.NewButton()
	ui.footer.findPrevBtn.SetSizeRequest(20, 20)
	ui.footer.findPrevBtn.Add(gtk.NewArrow(gtk.ARROW_UP, gtk.SHADOW_NONE))
	ui.footer.findPrevBtn.Clicked(ui.FindPrev)

	ui.footer.closeBtn = gtk.NewButton()
	ui.footer.closeBtn.SetSizeRequest(20, 20)
	ui.footer.closeBtn.Add(gtk.NewImageFromStock(gtk.STOCK_CLOSE, gtk.ICON_SIZE_BUTTON))
	ui.footer.closeBtn.Clicked(ui.FooterClose)
	ui.footer.closeBtn.AddAccelerator("activate", ui.accelGroup, gdk.KEY_Escape, 0, gtk.ACCEL_VISIBLE)

	// replacebar
	ui.footer.replEntry = gtk.NewEntryWithBuffer(gtk.NewEntryBuffer(""))
	// ui.footer.replEntry.Connect("changed", OnFindInput)

	ui.footer.replBtn = gtk.NewButton()
	ui.footer.replBtn.SetSizeRequest(20, 20)
	ui.footer.replBtn.Add(gtk.NewImageFromIconName("text-changelog", gtk.ICON_SIZE_BUTTON))
	ui.footer.replBtn.Clicked(ui.ReplaceOne)

	ui.footer.replAllBtn = gtk.NewButton()
	ui.footer.replAllBtn.SetSizeRequest(20, 20)
	ui.footer.replAllBtn.Add(gtk.NewImageFromIconName("text-plain", gtk.ICON_SIZE_BUTTON))
	ui.footer.replAllBtn.Clicked(ui.ReplaceAll)
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

func (ui *UI) ShowFindbar() {
	ui.footer.table.SetVisible(true)
	ui.footer.replEntry.SetVisible(false)
	ui.footer.replBtn.SetVisible(false)
	ui.footer.replAllBtn.SetVisible(false)

	ui.footer.findEntry.GrabFocus()
}

func (ui *UI) ShowReplbar() {
	ui.footer.table.SetVisible(true)
	ui.footer.replEntry.SetVisible(true)
	ui.footer.replBtn.SetVisible(true)
	ui.footer.replAllBtn.SetVisible(true)

	ui.footer.replEntry.GrabFocus()
}

func (ui *UI) FooterClose() {
	ui.footer.table.SetVisible(false)
}
