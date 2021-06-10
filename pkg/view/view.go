package view

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/fs/sync"
	"github.com/rivo/tview"
)

const (
	Local = "local"
)

type View struct {
	*tview.Table

	remotes       []string
	remote        string
	remotePath    []string
	remoteEntries fs.DirEntries

	status    *Status
	otherView *View
}

// renderHeader renders the header of the table.
// The table header always contains the name, size and date of a file/folder.
func (v *View) renderHeader() {
	v.SetCell(0, 0, tview.NewTableCell("NAME").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(5).SetSelectable(true))
	v.SetCell(0, 1, tview.NewTableCell("SIZE").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(2).SetSelectable(true))
	v.SetCell(0, 2, tview.NewTableCell("DATE").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(2).SetSelectable(true))
}

// renderRemotes renders the table which shows all configured remotes.
// Before we render the list of remotes we have to reset the selected remote, path and entries. Then we also clear the
// current view and status. After this we can render the header and each remote as a row.
func (v *View) renderRemotes(localPath []string) {
	v.remote = ""
	v.remotePath = nil
	v.remoteEntries = nil

	v.Clear()
	v.renderHeader()
	v.status.SetLocation("", nil)

	for i, remote := range v.remotes {
		v.SetCell(i+1, 0, tview.NewTableCell(remote).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 1, tview.NewTableCell("").SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 2, tview.NewTableCell("").SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
	}
}

// renderEntries renders the rows for all entries (files and folders) which are returned by rclone.
// When the user selected a file we do nothing. If the user selected a folder we add this folder to the list of paths
// and then we try to retrieve all entries for the new path and render them in the table.
func (v *View) renderEntries(app *tview.Application) {
	f, err := fs.NewFs(context.Background(), fsPath(v.remote, v.remotePath))
	if err != nil {
		if err == fs.ErrorIsFile {
			return
		} else {
			app.Stop()
			log.Fatalf("Could not create new fs object for \"%s\": %#v", fsPath(v.remote, v.remotePath), err)
		}
	}

	v.remoteEntries, err = f.List(context.Background(), "")
	if err != nil {
		app.Stop()
		log.Fatalf("Could not get entries for \"%s\": %#v", fsPath(v.remote, v.remotePath), err)
	}

	v.Clear()
	v.renderHeader()
	v.status.SetLocation(v.remote, v.remotePath)

	for i, entry := range v.remoteEntries {
		v.SetCell(i+1, 0, tview.NewTableCell(entry.String()).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%d", entry.Size())).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 2, tview.NewTableCell(entry.ModTime(context.Background()).Format("2006-01-02 15:04:05")).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
	}
}

// SetView is used to pass the other created view to this view instance. This is required so that we can switch the
// focus between views with the "tab" key.
func (v *View) SetView(otherView *View) {
	v.otherView = otherView
}

// NewView returns a new view. To create a new view we have to pass the app so that we can stop the application in case
// of an error. It also requires the status compnent, the remotes and the current directory of the user.
func NewView(app *tview.Application, status *Status, remotes, localPath []string) *View {
	v := &View{
		tview.NewTable().SetFixed(1, 0).SetSelectable(true, false).SetEvaluateAllRows(true).SetBorders(false),
		remotes,
		"",
		nil,
		nil,
		status,
		nil,
	}

	// We always show the list of remotes first.
	v.renderRemotes(localPath)

	// The following is used to handle a slection of an table row. A row can be selected by pressing "enter". This is
	// only used to navigate between folders. File actions are not triggered by "enter".
	v.SetSelectedFunc(func(row int, column int) {
		// When the first row is selected we do nothing, because this is always the table header.
		if row == 0 {
			return
		}

		// If no remote is set we are in the remotes view, where the user is able to select a remote or the special
		// local "remote", which will be the users current directory.
		// We also have to check if the user selected the special local "remote", because we have then also set the
		// path. This is only required, because the fsPath helper function would return an empty string if we do not do
		// that, which leads to some errors in the following steps.
		if v.remote == "" {
			if remotes[row-1] == Local {
				v.remotePath = localPath
			}

			v.remote = remotes[row-1]
		}

		// If the list of entries is larger then zero we can use the current selection to select an entry by the
		// provided row number (we have to substract 1, because of the header).
		// Before we adjust the path, we have to check if the user selected a file. If this is the case we do not modify
		// the current path.
		if len(v.remoteEntries) > 0 {
			entry := v.remoteEntries[row-1]

			_, err := fs.NewFs(context.Background(), fsPath(v.remote, append(v.remotePath, entry.String())))
			if err != nil {
				if err == fs.ErrorIsFile {
					return
				} else {
					app.Stop()
					log.Fatalf("Could not create new fs object for \"%s\": %#v", fsPath(v.remote, append(v.remotePath, entry.String())), err)
				}
			}

			v.remotePath = append(v.remotePath, entry.String())
		}

		v.renderEntries(app)
	})

	// We have to provide some additional navigation and action option. The default navigation keys can be found in the
	// tview documentation at

	// The following is used to register some custom key handlers. This is required so that we can provide additional
	// navigation and action features besides the default ones (https://pkg.go.dev/github.com/rivo/tview#hdr-Navigation)
	v.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// The "tab" key is used to switch the focus between our two view. For that we have to call the SetView function
		// right after both views were initialized.
		if event.Key() == tcell.KeyTAB {
			app.SetFocus(v.otherView)
		}

		// The "escape" key is used to go back to the remotes selection table. This allows a user to always escaped the
		// current entries table.
		if event.Key() == tcell.KeyEscape {
			v.renderRemotes(localPath)
		}

		// The "backspace" key is used to went up a directory. If there is no entry in the path list we go back to the
		// remotes selection table.
		if event.Key() == tcell.KeyBackspace2 && v.remote != "" {
			if len(v.remotePath) == 0 {
				v.renderRemotes(localPath)
			} else {
				v.remotePath = v.remotePath[:len(v.remotePath)-1]
				v.renderEntries(app)
			}
		}

		// The "c" key is used to copy the selected file. When the user presses the "c" key we add the current
		// file/folder as selected one. The selection is handled by the status component.
		// We have to check that the user does not selected the header column and that there are enough items in the
		// entries list for the selection.
		if event.Rune() == 'c' && v.remote != "" {
			row, _ := v.GetSelection()
			if row > 0 && row-1 < len(v.remoteEntries) {
				v.status.SetSelect(v.remote, append(v.remotePath, v.remoteEntries[row-1].String()), "copy")
			}
		}

		// The "p" key is used to paste the selected file/folder. When the user presses the "p" key and selected a file/
		// folder before with the "c" key the selected file/folder is paste in the current remote/path.
		// We have to check if the user already selected a remote and path. If this is the case we check if the source
		// entry is a file or a folder. If it is a file we have to remove the filename from the path and handle this in
		// an additional variable, so that we can use the operations.CopyFile function to copy the file from the source
		// (selected) to the destination (v.remote:v.remotePath). If the user selected a folder we can use the sync.Copy
		// function to copy the folder.
		if event.Rune() == 'p' && v.remote != "" && v.status.GetAction() == "copy" {
			selectedRemote := v.status.GetSelectedRemote()
			selectedPath := v.status.GetSelectedPath()

			if selectedRemote != "" && len(selectedPath) > 0 {
				v.status.SetSelect(selectedRemote, selectedPath, "paste")

				_, err := fs.NewFs(context.Background(), fsPath(selectedRemote, selectedPath))
				if err != nil {
					if err == fs.ErrorIsFile {
						path, filename := fsPathFilename(selectedPath)

						fsrc, err := fs.NewFs(context.Background(), fsPath(selectedRemote, path))
						if err != nil {
							app.Stop()
							log.Fatalf("Could not create new fsrc object: %#v", err)
						}

						fdst, err := fs.NewFs(context.Background(), fsPath(v.remote, v.remotePath))
						if err != nil {
							app.Stop()
							log.Fatalf("Could not create new fdst object: %#v", err)
						}

						err = operations.CopyFile(context.Background(), fdst, fsrc, filename, filename)
						if err != nil {
							app.Stop()
							log.Fatalf("Could not copy/paste file: %#v", err)
						}
					} else {
						app.Stop()
						log.Fatalf("Could not create new fs object for \"%s\": %#v", fsPath(selectedRemote, selectedPath), err)
					}
				} else {
					fsrc, err := fs.NewFs(context.Background(), fsPath(selectedRemote, selectedPath))
					if err != nil {
						app.Stop()
						log.Fatalf("Could not create new fsrc object: %#v", err)
					}

					fdst, err := fs.NewFs(context.Background(), fsPath(v.remote, append(v.remotePath, selectedPath[len(selectedPath)-1])))
					if err != nil {
						app.Stop()
						log.Fatalf("Could not create new fdst object: %#v", err)
					}

					err = sync.CopyDir(context.Background(), fdst, fsrc, true)
					if err != nil {
						app.Stop()
						log.Fatalf("Could not copy/paste folder: %#v", err)
					}
				}

				v.renderEntries(app)
			}

			v.status.SetSelect("", nil, "")
		}

		// The "d" key is used to delete the a file/folder. When the user presses the "d" key the first time the file/
		// folder is selected for delition. When the user presses the "d" key another time the selected file is deleted.
		// When the users presses another key in between, the selection is removed.
		if event.Rune() == 'd' && v.remote != "" {
			if v.status.GetAction() == "delete" {
				// User presses the "d" key the second time.
				selectedRemote := v.status.GetSelectedRemote()
				selectedPath := v.status.GetSelectedPath()

				if selectedRemote != "" && len(selectedPath) > 0 {
					if selectedRemote == Local {
						err := os.RemoveAll(fsPath(selectedRemote, selectedPath))
						if err != nil {
							app.Stop()
							log.Fatalf("Could not delete file: %#v", err)
						}
					} else {
						f, err := fs.NewFs(context.Background(), fsPath(selectedRemote, selectedPath))
						if err != nil {
							if err == fs.ErrorIsFile {
								path, filename := fsPathFilename(selectedPath)

								fdst, err := fs.NewFs(context.Background(), fsPath(selectedRemote, path))
								if err != nil {
									app.Stop()
									log.Fatalf("Could not create new fsrc object: %#v", err)
								}

								dst, err := fdst.NewObject(context.Background(), filename)
								if err != nil {
									app.Stop()
									log.Fatalf("Could not get file for deletion: %#v", err)
								}

								err = operations.DeleteFile(context.Background(), dst)
								if err != nil {
									app.Stop()
									log.Fatalf("Could not delete file: %#v", err)
								}
							} else {
								app.Stop()
								log.Fatalf("Could not create new fs object for \"%s\": %#v", fsPath(selectedRemote, selectedPath), err)
							}
						} else {
							err := operations.Delete(context.Background(), f)
							if err != nil {
								app.Stop()
								log.Fatalf("Could not delete folder: %#v", err)
							}
						}
					}
				}

				v.renderEntries(app)
				v.status.SetSelect("", nil, "")
			} else {
				// User presses the "d" key the first time.
				row, _ := v.GetSelection()
				if row > 0 && row-1 < len(v.remoteEntries) && len(v.remotePath) != 0 {
					v.status.SetSelect(v.remote, append(v.remotePath, v.remoteEntries[row-1].String()), "delete")
				}
			}
		} else {
			// User presses another key then "d".
			if v.status.GetAction() == "delete" {
				v.status.SetSelect("", nil, "")
			}
		}

		return event
	})

	return v
}
