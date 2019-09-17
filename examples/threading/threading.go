// Copyright 2011 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

type Foo struct {
	Index   int
	Bar     string
	Baz     float64
	Quux    time.Time
	checked bool
}

type FooModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	items      []*Foo
}

func NewFooModel() *FooModel {
	m := new(FooModel)
	m.ResetRows()
	return m
}

// Called by the TableView from SetModel and every time the model publishes a
// RowsReset event.
func (m *FooModel) RowCount() int {
	return len(m.items)
}

// Called by the TableView when it needs the text to display for a given cell.
func (m *FooModel) Value(row, col int) interface{} {
	item := m.items[row]

	switch col {
	case 0:
		return item.Index

	case 1:
		return item.Bar

	case 2:
		return item.Baz

	case 3:
		return item.Quux
	}

	panic("unexpected col")
}

// Called by the TableView to retrieve if a given row is checked.
func (m *FooModel) Checked(row int) bool {
	return m.items[row].checked
}

// Called by the TableView when the user toggled the check box of a given row.
func (m *FooModel) SetChecked(row int, checked bool) error {
	m.items[row].checked = checked

	return nil
}

// Called by the TableView to sort the model.
func (m *FooModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order

	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]

		c := func(ls bool) bool {
			if m.sortOrder == walk.SortAscending {
				return ls
			}

			return !ls
		}

		switch m.sortColumn {
		case 0:
			return c(a.Index < b.Index)

		case 1:
			return c(a.Bar < b.Bar)

		case 2:
			return c(a.Baz < b.Baz)

		case 3:
			return c(a.Quux.Before(b.Quux))
		}

		panic("unreachable")
	})

	return m.SorterBase.Sort(col, order)
}

func (m *FooModel) ResetRows() {
	// Create some random data.
	m.items = make([]*Foo, rand.Intn(50000))

	now := time.Now()

	for i := range m.items {
		m.items[i] = &Foo{
			Index: i,
			Bar:   strings.Repeat("*", rand.Intn(5)+1),
			Baz:   rand.Float64() * 1000,
			Quux:  time.Unix(rand.Int63n(now.Unix()), 0),
		}
	}

	// Notify TableView and other interested parties about the reset.
	m.PublishRowsReset()

	m.Sort(m.sortColumn, m.sortOrder)
}

func tableviewMain() {
	rand.Seed(time.Now().UnixNano())

	boldFont, _ := walk.NewFont("Segoe UI", 9, walk.FontBold)
	goodIcon, _ := walk.Resources.Icon("../img/check.ico")
	badIcon, _ := walk.Resources.Icon("../img/stop.ico")

	barBitmap, err := walk.NewBitmap(walk.Size{100, 1})
	if err != nil {
		panic(err)
	}
	defer barBitmap.Dispose()

	canvas, err := walk.NewCanvasFromImage(barBitmap)
	if err != nil {
		panic(err)
	}
	defer barBitmap.Dispose()

	canvas.GradientFillRectangle(walk.RGB(255, 0, 0), walk.RGB(0, 255, 0), walk.Horizontal, walk.Rectangle{0, 0, 100, 1})

	canvas.Dispose()

	model := NewFooModel()

	var tv *walk.TableView

	MainWindow{
		Title:  "Walk TableView Example",
		Size:   Size{800, 600},
		Layout: VBox{MarginsZero: true},
		Children: []Widget{
			PushButton{
				Text:      "Reset Rows",
				OnClicked: model.ResetRows,
			},
			PushButton{
				Text: "Select first 5 even Rows",
				OnClicked: func() {
					tv.SetSelectedIndexes([]int{0, 2, 4, 6, 8})
				},
			},
			TableView{
				AssignTo:         &tv,
				AlternatingRowBG: true,
				CheckBoxes:       true,
				ColumnsOrderable: true,
				MultiSelection:   true,
				Columns: []TableViewColumn{
					{Title: "#"},
					{Title: "Bar"},
					{Title: "Baz", Alignment: AlignFar},
					{Title: "Quux", Format: "2006-01-02 15:04:05", Width: 150},
				},
				StyleCell: func(style *walk.CellStyle) {
					item := model.items[style.Row()]

					if item.checked {
						if style.Row()%2 == 0 {
							style.BackgroundColor = walk.RGB(159, 215, 255)
						} else {
							style.BackgroundColor = walk.RGB(143, 199, 239)
						}
					}

					switch style.Col() {
					case 1:
						if canvas := style.Canvas(); canvas != nil {
							bounds := style.Bounds()
							bounds.X += 2
							bounds.Y += 2
							bounds.Width = int((float64(bounds.Width) - 4) / 5 * float64(len(item.Bar)))
							bounds.Height -= 4
							canvas.DrawBitmapPartWithOpacity(barBitmap, bounds, walk.Rectangle{0, 0, 100 / 5 * len(item.Bar), 1}, 127)

							bounds.X += 4
							bounds.Y += 2
							canvas.DrawText(item.Bar, tv.Font(), 0, bounds, walk.TextLeft)
						}

					case 2:
						if item.Baz >= 900.0 {
							style.TextColor = walk.RGB(0, 191, 0)
							style.Image = goodIcon
						} else if item.Baz < 100.0 {
							style.TextColor = walk.RGB(255, 0, 0)
							style.Image = badIcon
						}

					case 3:
						if item.Quux.After(time.Now().Add(-365 * 24 * time.Hour)) {
							style.Font = boldFont
						}
					}
				},
				Model: model,
				OnSelectedIndexesChanged: func() {
					fmt.Printf("SelectedIndexes: %v\n", tv.SelectedIndexes())
				},
			},
		},
	}.Run()
}

var isSpecialMode = walk.NewMutableCondition()

type MyMainWindow struct {
	*walk.MainWindow
}

func actionsMain() {
	MustRegisterCondition("isSpecialMode", isSpecialMode)

	mw := new(MyMainWindow)

	var openAction, showAboutBoxAction *walk.Action
	var recentMenu *walk.Menu
	var toggleSpecialModePB *walk.PushButton

	if err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "Walk Actions Example",
		MenuItems: []MenuItem{
			Menu{
				Text: "&File",
				Items: []MenuItem{
					Action{
						AssignTo:    &openAction,
						Text:        "&Open",
						Image:       "../img/open.png",
						Enabled:     Bind("enabledCB.Checked"),
						Visible:     Bind("!openHiddenCB.Checked"),
						Shortcut:    Shortcut{walk.ModControl, walk.KeyO},
						OnTriggered: mw.openAction_Triggered,
					},
					Menu{
						AssignTo: &recentMenu,
						Text:     "Recent",
					},
					Separator{},
					Action{
						Text:        "E&xit",
						OnTriggered: func() { mw.Close() },
					},
				},
			},
			Menu{
				Text: "&View",
				Items: []MenuItem{
					Action{
						Text:    "Open / Special Enabled",
						Checked: Bind("enabledCB.Visible"),
					},
					Action{
						Text:    "Open Hidden",
						Checked: Bind("openHiddenCB.Visible"),
					},
				},
			},
			Menu{
				Text: "&Help",
				Items: []MenuItem{
					Action{
						AssignTo:    &showAboutBoxAction,
						Text:        "About",
						OnTriggered: mw.showAboutBoxAction_Triggered,
					},
				},
			},
		},
		ToolBar: ToolBar{
			ButtonStyle: ToolBarButtonImageBeforeText,
			Items: []MenuItem{
				ActionRef{&openAction},
				Menu{
					Text:  "New A",
					Image: "../img/document-new.png",
					Items: []MenuItem{
						Action{
							Text:        "A",
							OnTriggered: mw.newAction_Triggered,
						},
						Action{
							Text:        "B",
							OnTriggered: mw.newAction_Triggered,
						},
						Action{
							Text:        "C",
							OnTriggered: mw.newAction_Triggered,
						},
					},
					OnTriggered: mw.newAction_Triggered,
				},
				Separator{},
				Menu{
					Text:  "View",
					Image: "../img/document-properties.png",
					Items: []MenuItem{
						Action{
							Text:        "X",
							OnTriggered: mw.changeViewAction_Triggered,
						},
						Action{
							Text:        "Y",
							OnTriggered: mw.changeViewAction_Triggered,
						},
						Action{
							Text:        "Z",
							OnTriggered: mw.changeViewAction_Triggered,
						},
					},
				},
				Separator{},
				Action{
					Text:        "Special",
					Image:       "../img/system-shutdown.png",
					Enabled:     Bind("isSpecialMode && enabledCB.Checked"),
					OnTriggered: mw.specialAction_Triggered,
				},
			},
		},
		ContextMenuItems: []MenuItem{
			ActionRef{&showAboutBoxAction},
		},
		MinSize: Size{300, 200},
		Layout:  VBox{},
		Children: []Widget{
			CheckBox{
				Name:    "enabledCB",
				Text:    "Open / Special Enabled",
				Checked: true,
			},
			CheckBox{
				Name:    "openHiddenCB",
				Text:    "Open Hidden",
				Checked: true,
			},
			PushButton{
				AssignTo: &toggleSpecialModePB,
				Text:     "Enable Special Mode",
				OnClicked: func() {
					isSpecialMode.SetSatisfied(!isSpecialMode.Satisfied())

					if isSpecialMode.Satisfied() {
						toggleSpecialModePB.SetText("Disable Special Mode")
					} else {
						toggleSpecialModePB.SetText("Enable Special Mode")
					}
				},
			},
		},
	}.Create()); err != nil {
		log.Fatal(err)
	}

	addRecentFileActions := func(texts ...string) {
		for _, text := range texts {
			a := walk.NewAction()
			a.SetText(text)
			a.Triggered().Attach(mw.openAction_Triggered)
			recentMenu.Actions().Add(a)
		}
	}

	addRecentFileActions("Foo", "Bar", "Baz")

	mw.Run()
}

func (mw *MyMainWindow) openAction_Triggered() {
	walk.MsgBox(mw, "Open", "Pretend to open a file...", walk.MsgBoxIconInformation)
}

func (mw *MyMainWindow) newAction_Triggered() {
	walk.MsgBox(mw, "New", "Newing something up... or not.", walk.MsgBoxIconInformation)
}

func (mw *MyMainWindow) changeViewAction_Triggered() {
	walk.MsgBox(mw, "Change View", "By now you may have guessed it. Nothing changed.", walk.MsgBoxIconInformation)
}

func (mw *MyMainWindow) showAboutBoxAction_Triggered() {
	walk.MsgBox(mw, "About", "Walk Actions Example", walk.MsgBoxIconInformation)
}

func (mw *MyMainWindow) specialAction_Triggered() {
	walk.MsgBox(mw, "Special", "Nothing to see here.", walk.MsgBoxIconInformation)
}

func main() {
	var wg sync.WaitGroup
	// to set up a wndproc
	walk.Init(true)
	walk.Threaded = true
	wg.Add(2)
	go func() {
		walk.EnterThread()
		defer walk.LeaveThread()
		actionsMain()
		wg.Done()
	}()
	go func() {
		walk.EnterThread()
		defer walk.LeaveThread()
		tableviewMain()
		wg.Done()
	}()
	wg.Wait()
}
