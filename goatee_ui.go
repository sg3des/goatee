package main

import (
	"log"
	"path"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
)

type UI struct {
	window *gtk.Window

	accelGroup  *gtk.AccelGroup
	actionGroup *gtk.ActionGroup

	vbox     *gtk.VBox
	menubar  *gtk.Widget
	notebook *gtk.Notebook
	tabs     []*Tab

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
	ui.window.SetSizeRequest(100, 100)

	ui.vbox = gtk.NewVBox(false, 0)
	ui.vbox.PackStart(ui.createUIManager(), false, false, 0)

	ui.notebook = gtk.NewNotebook()
	ui.vbox.PackStart(ui.notebook, true, true, 0)
	ui.vbox.PackStart(ui.createFooter(), false, false, 0)
	ui.window.Add(ui.vbox)

	ui.window.Connect("destroy", ui.Quit)
	if conf.Tabs.Homogenous {
		ui.window.Connect("check-resize", ui.homogeneousTabs)
	}

	ui.window.ShowAll()

	ui.footer.table.SetVisible(false)
	ui.menubar.SetVisible(conf.UI.MenuBarVisible)
	return ui
}

func (ui *UI) Open() {
	dialog := gtk.NewFileChooserDialog("Open File", ui.window, gtk.FILE_CHOOSER_ACTION_OPEN, gtk.STOCK_CANCEL, gtk.RESPONSE_CANCEL, gtk.STOCK_OPEN, gtk.RESPONSE_ACCEPT)

	if dialog.Run() == gtk.RESPONSE_ACCEPT {
		ui.NewTab(dialog.GetFilename())
	}
	dialog.Destroy()
}
func (ui *UI) Save() {
	t := ui.GetCurrentTab()

	if len(t.Filename) == 0 {
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
	t := ui.GetCurrentTab()

	filename := dialogSave()
	if len(filename) == 0 {
		return
	}

	t.Filename = filename
	t.label.SetText(path.Base(filename))
	t.label.SetTooltipText(filename)
	t.Save()
}

func (ui *UI) Quit() {
	for _, t := range ui.tabs {
		t.File.Close()
	}
	gtk.MainQuit()
}

func (ui *UI) Find() {
	ui.GetCurrentTab().Find()
}
func (ui *UI) FindNext() {
	ui.GetCurrentTab().FindNext(true)
}
func (ui *UI) FindPrev() {
	ui.GetCurrentTab().FindNext(false)
}
func (ui *UI) ReplaceOne() {
	ui.GetCurrentTab().Replace(false)
}
func (ui *UI) ReplaceAll() {
	ui.GetCurrentTab().Replace(true)
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
	dialog := gtk.NewFileChooserDialog("Save File", ui.window, gtk.FILE_CHOOSER_ACTION_SAVE, gtk.STOCK_CANCEL, gtk.RESPONSE_CANCEL, gtk.STOCK_SAVE, gtk.RESPONSE_ACCEPT)

	var filename string
	if dialog.Run() == gtk.RESPONSE_ACCEPT {
		filename = dialog.GetFilename()
	}

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
			<separator />
			<menuitem action='Preferences'/>
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

	ui.newAction("NewTab", "New Tab", "<control>t", func() { ui.NewTab("") })
	ui.newActionStock("Open", gtk.STOCK_OPEN, "", ui.Open)
	ui.newActionStock("Save", gtk.STOCK_SAVE, "", ui.Save)
	ui.newActionStock("SaveAs", gtk.STOCK_SAVE_AS, "<control><shift>s", ui.SaveAs)
	ui.newAction("CloseTab", "Close Tab", "<control>w", ui.CloseCurrentTab)
	ui.newActionStock("Quit", gtk.STOCK_QUIT, "", ui.Quit)

	// Edit
	ui.actionGroup.AddAction(gtk.NewAction("Edit", "Edit", "", ""))

	ui.newActionStock("Find", gtk.STOCK_FIND, "", ui.ShowFindbar)
	ui.newAction("FindNext", "Find Next", "F3", ui.FindNext)
	ui.newAction("FindPrev", "Find Previous", "<shift>F3", ui.FindPrev)

	ui.newActionStock("Replace", gtk.STOCK_FIND_AND_REPLACE, "<control>h", ui.ShowReplbar)
	ui.newAction("ReplaceOne", "Replace One", "<control><shift>h", ui.ReplaceOne)
	ui.newAction("ReplaceAll", "Replace All", "<control><alt>Return", ui.ReplaceAll)
	ui.newAction("Preferences", "Preferences", "<control><shift>p", conf.OpenWindow)

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

func (ui *UI) homogeneousTabs() {
	if len(ui.tabs) == 0 || !conf.Tabs.Homogenous {
		return
	}

	var width, height int
	ui.window.GetSize(&width, &height)

	tabwidth := (width - len(ui.tabs)*6) / len(ui.tabs)
	leftwidth := (width - len(ui.tabs)*6) % len(ui.tabs)

	for _, t := range ui.tabs {
		if leftwidth > 0 {
			t.tabbox.SetSizeRequest(tabwidth+1, conf.Tabs.Height)
			leftwidth--
		} else {
			t.tabbox.SetSizeRequest(tabwidth, conf.Tabs.Height)
		}
	}
}

func (ui *UI) TabsContains(filename string) bool {
	for n, t := range ui.tabs {
		if t.Filename == filename {
			ui.notebook.SetCurrentPage(n)
			return true
		}
	}
	return false
}

func (ui *UI) CloseCurrentTab() {
	n := ui.notebook.GetCurrentPage()
	ui.CloseTab(n)
}

func (ui *UI) CloseTab(n int) {
	ui.tabs[n].Close()

	if len(ui.tabs) == 0 {
		gtk.MainQuit()
	}
}

func (ui *UI) GetCurrentTab() *Tab {
	n := ui.notebook.GetCurrentPage()
	if n < 0 {
		return nil
	}
	return ui.tabs[n]
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
	for _, t := range ui.tabs {
		t.ClearTags()
	}
	ui.footer.table.SetVisible(false)
}
