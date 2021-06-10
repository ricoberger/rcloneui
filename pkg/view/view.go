package view

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/otiai10/copy"
	rclonefs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/fs/sync"
	"github.com/rivo/tview"
)

const (
	Local = "local"
)

type View struct {
	*tview.Table

	remotes []string
	remote  string

	remotePath    []string
	remoteEntries rclonefs.DirEntries

	localPath    []string
	localEntries []fs.FileInfo

	status    *Status
	otherView *View
}

func (v *View) showRemotes(localPath []string) {
	v.remote = ""
	v.remotePath = nil
	v.remoteEntries = nil
	v.localPath = localPath
	v.localEntries = nil

	v.Clear()
	v.createHeader()
	v.status.SetLocation("", nil)

	for i, remote := range v.remotes {
		v.SetCell(i+1, 0, tview.NewTableCell(remote).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 1, tview.NewTableCell("").SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 2, tview.NewTableCell("").SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
	}
}

func (v *View) createHeader() {
	v.SetCell(0, 0, tview.NewTableCell("NAME").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(5).SetSelectable(true))
	v.SetCell(0, 1, tview.NewTableCell("SIZE").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(2).SetSelectable(true))
	v.SetCell(0, 2, tview.NewTableCell("DATE").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft).SetExpansion(2).SetSelectable(true))
}

func (v *View) renderRemote(app *tview.Application) {
	f, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", v.remote, strings.Join(v.remotePath, "/")))
	if err != nil {
		if err == rclonefs.ErrorIsFile {
			// When the user selected a file we do nothing. The user has to use one of the keys to trigger an action.
			return
		} else {
			app.Stop()
			log.Fatalf("Could not create new fs object for \"%s:%s\": %#v", v.remote, strings.Join(v.remotePath, "/"), err)
		}
	}

	v.remoteEntries, err = f.List(context.Background(), "")
	if err != nil {
		app.Stop()
		log.Fatalf("Could not get entries for \"%s:%s\": %#v", v.remote, strings.Join(v.remotePath, "/"), err)
	}

	v.Clear()
	v.createHeader()
	v.status.SetLocation(v.remote, v.remotePath)

	for i, entry := range v.remoteEntries {
		v.SetCell(i+1, 0, tview.NewTableCell(entry.String()).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%d", entry.Size())).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 2, tview.NewTableCell(entry.ModTime(context.Background()).Format("2006-01-02 15:04:05")).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
	}
}

func (v *View) renderLocal(app *tview.Application) {
	var err error

	v.localEntries, err = ioutil.ReadDir(path.Join("/", path.Join(v.localPath...)))
	if err != nil {
		app.Stop()
		log.Fatalf("Could not list files: %#v", err)
	}

	v.Clear()
	v.createHeader()
	v.status.SetLocation(v.remote, v.localPath)

	for i, entry := range v.localEntries {
		v.SetCell(i+1, 0, tview.NewTableCell(entry.Name()).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%d", entry.Size())).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
		v.SetCell(i+1, 2, tview.NewTableCell(entry.ModTime().Format("2006-01-02 15:04:05")).SetTextColor(tcell.ColorBlue).SetAlign(tview.AlignLeft))
	}
}

func (v *View) SetView(otherView *View) {
	v.otherView = otherView
}

func NewView(app *tview.Application, status *Status, remotes []string, userDir string) *View {
	v := &View{
		tview.NewTable().SetFixed(1, 0).SetSelectable(true, false).SetEvaluateAllRows(true).SetBorders(false),
		remotes,
		"",
		nil,
		nil,
		strings.Split(userDir, "/"),
		nil,
		status,
		nil,
	}

	v.showRemotes(strings.Split(userDir, "/"))

	v.SetSelectedFunc(func(row int, column int) {
		// When the first row is selected we do nothing, because this is always the table header.
		if row == 0 {
			return
		}

		// If no remote is set we are in the remotes view, where the user is able to select a remote or the special
		// local "remote", which will be the users current local directory.
		if v.remote == "" {
			v.remote = remotes[row-1]
		}

		// The handling for the local remote and all other remotes differes how we get the files and directories, so
		// that we have to check if the users uses the local remote.
		if v.remote == Local {
			// When the list of local entries is not empty, the user already made a minimum of one selection in the
			// local remote, so that we set this selection as the new one.
			if len(v.localEntries) > 0 {
				entry := v.localEntries[row-1]

				// If the selected entry is a directory we show the content of the directory next.
				if entry.IsDir() {
					v.localPath = append(v.localPath, entry.Name())
				} else {
					// When the user selected a file we do nothing. The user has to use one of the keys to trigger an
					// action.
					return
				}
			}

			v.renderLocal(app)
		} else {
			if len(v.remoteEntries) > 0 {
				entry := v.remoteEntries[row-1]
				v.remotePath = append(v.remotePath, entry.String())
			}

			v.renderRemote(app)
		}
	})

	// We have to provide some additional navigation and action option. The default navigation keys can be found in the
	// tview documentation at https://pkg.go.dev/github.com/rivo/tview#hdr-Navigation.
	v.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// The "tab" key we switch the focus to the other view.
		if event.Key() == tcell.KeyTAB {
			app.SetFocus(v.otherView)
		}

		// The "excape" key is used to go back to the remotes selection screen.
		if event.Key() == tcell.KeyEscape {
			v.showRemotes(strings.Split(userDir, "/"))
		}

		// The "backspace" key is used to went up a directory.
		if event.Key() == tcell.KeyBackspace2 && v.remote != "" {
			if v.remote == Local && len(v.localPath) != 0 {
				v.localPath = v.localPath[:len(v.localPath)-1]
				v.renderLocal(app)
			} else if len(v.remotePath) != 0 {
				v.remotePath = v.remotePath[:len(v.remotePath)-1]
				v.renderRemote(app)
			}
		}

		// The "c" key is used to copy the selected file.
		if event.Rune() == 'c' && v.remote != "" {
			row, _ := v.GetSelection()
			if v.remote == Local {
				v.status.SetSelect(v.remote, append(v.localPath, v.localEntries[row-1].Name()), "copy")
			} else if len(v.remotePath) != 0 {
				v.status.SetSelect(v.remote, append(v.remotePath, v.remoteEntries[row-1].String()), "copy")
			}
		}

		// The "p" key is used to paste the selected file.
		if event.Rune() == 'p' && v.remote != "" {
			selectedRemote := v.status.GetSelectedRemote()
			selectedPath := v.status.GetSelectedPath()

			v.status.SetSelect(selectedRemote, selectedPath, "paste")

			if v.remote == Local {
				if selectedRemote == Local {
					// Copy the file/folder from a local source to a local destination.
					// We are using the github.com/otiai10/copy package for that, so that we have not to handle if the
					// user selected a file or directory. When the file was successfully copied we rerender the local
					// directory.
					err := copy.Copy(path.Join("/", path.Join(selectedPath...)), path.Join("/", path.Join(v.localPath...), "/", selectedPath[len(selectedPath)-1]))
					if err != nil {
						app.Stop()
						log.Fatalf("Could not copy/paste file/folder: %#v", err)
					}

					v.renderLocal(app)
				} else {
					// Copy the file/folder from remote source to a local destionation.
					// In the first step we have to check if the selected entry is a file or folder. If it is a file we
					// are using the operations.CopyFile function to copy the file to the local destination. If the
					// entry is a folder we are using the sync.CopyDir function to copy the folder to the local
					// destination.
					_, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", selectedRemote, strings.Join(selectedPath, "/")))
					if err != nil {
						if err == rclonefs.ErrorIsFile {
							fsrc, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", selectedRemote, strings.Join(selectedPath[:len(selectedPath)-1], "/")))
							if err != nil {
								app.Stop()
								log.Fatalf("Could not create new fsrc object: %#v", err)
							}

							fdst, err := rclonefs.NewFs(context.Background(), path.Join("/", path.Join(v.localPath...)))
							if err != nil {
								app.Stop()
								log.Fatalf("Could not create new fdst object: %#v", err)
							}

							filename := selectedPath[len(selectedPath)-1]
							err = operations.CopyFile(context.Background(), fdst, fsrc, filename, filename)
							if err != nil {
								app.Stop()
								log.Fatalf("Could not copy/paste file: %#v", err)
							}
						} else {
							app.Stop()
							log.Fatalf("Could not create new fs object for \"%s:%s\": %#v", selectedRemote, strings.Join(selectedPath, "/"), err)
						}
					} else {
						fsrc, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", selectedRemote, strings.Join(selectedPath[:len(selectedPath)-1], "/")))
						if err != nil {
							app.Stop()
							log.Fatalf("Could not create new fsrc object: %#v", err)
						}

						fdst, err := rclonefs.NewFs(context.Background(), path.Join("/", path.Join(v.localPath...)))
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

					v.renderLocal(app)
				}
			} else {
				if selectedRemote == Local {
					// Copy file from a local source to a remote destination.
					// In the first step we have to check if the selected entry is a file or folder. If it is a file we
					// are using the operations.CopyFile function to copy the file to the local destination. If the
					// entry is a folder we are using the sync.CopyDir function to copy the folder to the local
					// destination.
					_, err := rclonefs.NewFs(context.Background(), strings.Join(selectedPath, "/"))
					if err != nil {
						if err == rclonefs.ErrorIsFile {
							fsrc, err := rclonefs.NewFs(context.Background(), path.Join("/", path.Join(selectedPath[:len(selectedPath)-1]...)))
							if err != nil {
								app.Stop()
								log.Fatalf("Could not create new fsrc object: %#v", err)
							}

							fdst, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", v.remote, strings.Join(v.remotePath, "/")))
							if err != nil {
								app.Stop()
								log.Fatalf("Could not create new fdst object: %#v", err)
							}

							filename := selectedPath[len(selectedPath)-1]
							err = operations.CopyFile(context.Background(), fdst, fsrc, filename, filename)
							if err != nil {
								app.Stop()
								log.Fatalf("Could not copy/paste file: %#v", err)
							}
						} else {
							app.Stop()
							log.Fatalf("Could not create new fs object for \"%s:%s\": %#v", selectedRemote, strings.Join(selectedPath, "/"), err)
						}
					} else {
						fsrc, err := rclonefs.NewFs(context.Background(), path.Join("/", path.Join(selectedPath...)))
						if err != nil {
							app.Stop()
							log.Fatalf("Could not create new fsrc object: %#v", err)
						}

						folder := strings.Join(v.remotePath, "/") + "/" + selectedPath[len(selectedPath)-1]
						fdst, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", v.remote, folder))
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

					v.renderRemote(app)
				} else {
					// Copy file from a remote source to a remote destination.
					// In the first step we have to check if the selected entry is a file or folder. If it is a file we
					// are using the operations.CopyFile function to copy the file to the local destination. If the
					// entry is a folder we are using the sync.CopyDir function to copy the folder to the local
					// destination.
					_, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", selectedRemote, strings.Join(selectedPath, "/")))
					if err != nil {
						if err == rclonefs.ErrorIsFile {
							fsrc, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", selectedRemote, strings.Join(selectedPath[:len(selectedPath)-1], "/")))
							if err != nil {
								app.Stop()
								log.Fatalf("Could not create new fsrc object: %#v", err)
							}

							fdst, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", v.remote, strings.Join(v.remotePath, "/")))
							if err != nil {
								app.Stop()
								log.Fatalf("Could not create new fdst object: %#v", err)
							}

							filename := selectedPath[len(selectedPath)-1]
							err = operations.CopyFile(context.Background(), fdst, fsrc, filename, filename)
							if err != nil {
								app.Stop()
								log.Fatalf("Could not copy/paste file: %#v", err)
							}
						} else {
							app.Stop()
							log.Fatalf("Could not create new fs object for \"%s:%s\": %#v", selectedRemote, strings.Join(selectedPath, "/"), err)
						}
					} else {
						fsrc, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", selectedRemote, strings.Join(selectedPath, "/")))
						if err != nil {
							app.Stop()
							log.Fatalf("Could not create new fsrc object: %#v", err)
						}

						folder := strings.Join(v.remotePath, "/") + "/" + selectedPath[len(selectedPath)-1]
						fdst, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", v.remote, folder))
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

					v.renderRemote(app)
				}
			}

			v.status.SetSelect("", nil, "")
		}

		// The "d" key is used to delete the selected file.
		if event.Rune() == 'd' && v.remote != "" {
			// If "d" was already pressed one time we delete the file/folder. For that we have to check if the file is
			// on the local remote or an other remote.
			if v.status.GetAction() == "delete" {
				selectedRemote := v.status.GetSelectedRemote()
				selectedPath := v.status.GetSelectedPath()

				// For a local remote we are using os.RemoveAll to remove the file/folder.
				if selectedRemote == Local {
					if len(selectedPath) > 0 {
						err := os.RemoveAll(path.Join("/", path.Join(selectedPath...)))
						if err != nil {
							app.Stop()
							log.Fatalf("Could not delete file: %#v", err)
						}

						v.renderLocal(app)
					}
				} else {
					// For all other remotes we have to check if the selection is a file or folder. So that we can use
					// the corresponding functions operations.DeleteFile or operations.Delete.
					if len(selectedPath) > 0 {
						f, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", selectedRemote, strings.Join(selectedPath, "/")))
						if err != nil {
							if err == rclonefs.ErrorIsFile {
								fdst, err := rclonefs.NewFs(context.Background(), fmt.Sprintf("%s:%s", selectedRemote, strings.Join(selectedPath[:len(selectedPath)-1], "/")))
								if err != nil {
									log.Fatalf("Could not create new fsrc object: %#v", err)
								}

								dst, err := fdst.NewObject(context.Background(), selectedPath[len(selectedPath)-1])
								if err != nil {
									log.Fatalf("Could not get file for deletion: %#v", err)
								}

								err = operations.DeleteFile(context.Background(), dst)
								if err != nil {
									log.Fatalf("Could not delete file: %#v", err)
								}
							} else {
								app.Stop()
								log.Fatalf("Could not create new fs object for \"%s:%s\": %#v", selectedRemote, strings.Join(selectedPath, "/"), err)
							}
						} else {
							err := operations.Delete(context.Background(), f)
							if err != nil {
								log.Fatalf("Could not delete folder: %#v", err)
							}
						}

						v.renderRemote(app)
					}
				}

				v.status.SetSelect("", nil, "")
			} else {
				// If "d" is pressed the first time we initialize the delete action, by setting the remote and path as
				// selection.
				row, _ := v.GetSelection()
				if v.remote == Local {
					if len(v.localEntries) > 0 {
						v.status.SetSelect(v.remote, append(v.localPath, v.localEntries[row-1].Name()), "delete")
					}
				} else {
					if len(v.remotePath) != 0 {
						v.status.SetSelect(v.remote, append(v.remotePath, v.remoteEntries[row-1].String()), "delete")
					}
				}
			}
		} else {
			// When the user presses another key as "d" after "d" was pressed the first time we clear the selection.
			// This should avoid that a user accidently deletes a file by pressing "d".
			if v.status.GetAction() == "delete" {
				v.status.SetSelect("", nil, "")
			}
		}

		return event
	})

	return v
}
