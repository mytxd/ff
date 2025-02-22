package gui

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell"
	"github.com/skanehira/ff/system"
)

var (
	ErrNoDirName       = errors.New("no directory name")
	ErrNoFileName      = errors.New("no file name")
	ErrNoFileOrDirName = errors.New("no file or directory name")
	ErrNoFileOrDir     = errors.New("no file or directory")
	ErrNoNewName       = errors.New("no new name")
)

func (gui *Gui) SetKeybindings() {
	gui.InputPathKeybinding()
	gui.EntryManagerKeybinding()
}

// globalKeybinding
func (gui *Gui) GlobalKeybinding(event *tcell.EventKey) {
	switch {
	// go to input view
	case event.Key() == tcell.KeyTab:
		gui.App.SetFocus(gui.InputPath)

	// go to previous history
	//case event.Key() == tcell.KeyCtrlH:
	//	history := gui.HistoryManager.Previous()
	//	if history != nil {
	//		gui.InputPath.SetText(history.Path)
	//		gui.EntryManager.SetEntries(history.Path)
	//		gui.EntryManager.Select(history.RowIdx, 0)
	//	}

	//// go to next history
	//case event.Key() == tcell.KeyCtrlL:
	//	history := gui.HistoryManager.Next()
	//	if history != nil {
	//		gui.InputPath.SetText(history.Path)
	//		gui.EntryManager.SetEntries(history.Path)
	//		gui.EntryManager.Select(history.RowIdx, 0)
	//	}

	// go to parent dir
	case event.Rune() == 'h':
		current := gui.InputPath.GetText()
		parent := filepath.Dir(current)

		if parent != "" {
			// save select position
			gui.EntryManager.SetSelectPos(current)

			// update entries
			gui.InputPath.SetText(parent)
			gui.EntryManager.SetEntries(parent)
			gui.EntryManager.SetOffset(0, 0)

			// restore select position
			gui.EntryManager.RestorePos(parent)

			if gui.enablePreview {
				entry := gui.EntryManager.GetSelectEntry()
				gui.Preview.UpdateView(gui, entry)
			}
		}

	// go to selected dir
	case event.Rune() == 'l':
		entry := gui.EntryManager.GetSelectEntry()

		if entry != nil && entry.IsDir {
			// save select position
			gui.EntryManager.SetSelectPos(gui.InputPath.GetText())
			gui.EntryManager.SetEntries(entry.PathName)

			gui.InputPath.SetText(entry.PathName)

			gui.EntryManager.RestorePos(entry.PathName)

			row, _ := gui.EntryManager.GetSelection()
			count := gui.EntryManager.GetRowCount()
			if row > count {
				gui.EntryManager.Select(count-1, 0)
			}

			if gui.enablePreview {
				entry := gui.EntryManager.GetSelectEntry()
				gui.Preview.UpdateView(gui, entry)
			}
		}
	}
}

func (gui *Gui) EntryManagerKeybinding() {
	gui.EntryManager.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			gui.App.Stop()
		}

	}).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		gui.GlobalKeybinding(event)
		switch event.Rune() {
		// cut entry
		case 'd':
			if !hasEntry(gui) {
				return event
			}

			gui.Confirm("do you want to remove this?", "yes", gui.EntryManager, func() error {
				entry := gui.EntryManager.GetSelectEntry()
				if entry == nil {
					return nil
				}

				if entry.IsDir {
					if err := system.RemoveDirAll(entry.PathName); err != nil {
						log.Println(err)
						return err
					}
				} else {
					if err := system.RemoveFile(entry.PathName); err != nil {
						log.Println(err)
						return err
					}
				}

				path := gui.InputPath.GetText()
				gui.EntryManager.SetEntries(path)
				return nil
			})

		// copy entry
		case 'y':
			if !hasEntry(gui) {
				return event
			}

			m := gui.EntryManager
			m.UpdateColor()
			entry := m.GetSelectEntry()
			gui.Register.CopySource = entry

			row, _ := m.GetSelection()
			for i := 0; i < 5; i++ {
				m.GetCell(row, i).SetTextColor(tcell.ColorYellow)
			}

		// paste entry
		case 'p':
			source := gui.Register.CopySource

			gui.Form(map[string]string{"name": source.Name}, "paste", "new name", "new_name", gui.EntryManager,
				7, func(values map[string]string) error {
					name := values["name"]
					if name == "" {
						return ErrNoNewName
					}

					target := filepath.Join(gui.InputPath.GetText(), name)
					if err := system.CopyFile(source.PathName, target); err != nil {
						log.Println(err)
						return err
					}

					gui.EntryManager.SetEntries(gui.InputPath.GetText())
					return nil
				})

		// edit file with $EDITOR
		case 'e':
			editor := os.Getenv("EDITOR")
			if editor == "" {
				log.Println("$EDITOR is empty, please set $EDITOR")
				return event
			}

			entry := gui.EntryManager.GetSelectEntry()
			if entry == nil {
				log.Println("cannot get entry")
				return event
			}

			gui.App.Suspend(func() {
				if err := gui.ExecCmd(true, editor, entry.PathName); err != nil {
					log.Printf("%s: %s\n", ErrEdit, err)
				}
			})

			if gui.enablePreview {
				entry := gui.EntryManager.GetSelectEntry()
				gui.Preview.UpdateView(gui, entry)
			}
		case 'm':
			gui.Form(map[string]string{"name": ""}, "create", "new direcotry",
				"create_directory", gui.EntryManager,
				7, func(values map[string]string) error {
					name := values["name"]
					if name == "" {
						return ErrNoDirName
					}

					target := filepath.Join(gui.InputPath.GetText(), name)
					if err := system.NewDir(target); err != nil {
						log.Println(err)
						return err
					}

					gui.EntryManager.SetEntries(gui.InputPath.GetText())
					return nil
				})
		case 'r':
			gui.Form(map[string]string{"new name": ""}, "rename", "new name", "rename", gui.EntryManager,
				7, func(values map[string]string) error {
					name := values["new name"]
					if name == "" {
						return ErrNoFileName
					}

					current := gui.InputPath.GetText()

					entry := gui.EntryManager.GetSelectEntry()
					if entry == nil {
						return ErrNoFileOrDir
					}

					target := filepath.Join(current, name)
					if err := system.Rename(entry.PathName, target); err != nil {
						return err
					}

					gui.EntryManager.SetEntries(gui.InputPath.GetText())
					return nil
				})

		case 'n':
			gui.Form(map[string]string{"name": ""}, "create", "new file", "create_file", gui.EntryManager,
				7, func(values map[string]string) error {
					name := values["name"]
					if name == "" {
						return ErrNoFileOrDirName
					}

					target := filepath.Join(gui.InputPath.GetText(), name)
					if err := system.NewFile(target); err != nil {
						log.Println(err)
						return err
					}

					gui.EntryManager.SetEntries(gui.InputPath.GetText())
					return nil
				})
		case 'q':
			gui.Stop()
		}

		return event
	})

	gui.EntryManager.SetSelectionChangedFunc(func(row, col int) {
		if row > 0 {
			if gui.enablePreview {
				f := gui.EntryManager.Entries()[row-1]
				gui.Preview.UpdateView(gui, f)
			}
		}
	})

}

func (gui *Gui) InputPathKeybinding() {
	gui.InputPath.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			gui.App.Stop()
		}

		if key == tcell.KeyEnter {
			path := gui.InputPath.GetText()
			path = os.ExpandEnv(path)
			gui.InputPath.SetText(path)
			row, _ := gui.EntryManager.GetSelection()
			gui.HistoryManager.Save(row, path)
			gui.EntryManager.SetEntries(path)
		}

	}).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			gui.App.SetFocus(gui.EntryManager)
		}

		return event
	})
}
