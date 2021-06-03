package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ricoberger/rcloneui/pkg/prompt"
	"github.com/ricoberger/rcloneui/pkg/version"

	"github.com/manifoldco/promptui"
	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/rclone/rclone/fs/operations"
	flag "github.com/spf13/pflag"
)

var (
	showVersion bool
)

// init is used to define all flags for rcloneui. For example we define the --version flag here, which can be used to
// print the version information of rcloneui.
func init() {
	flag.BoolVar(&showVersion, "version", false, "Print version information.")
}

func main() {
	flag.Parse()

	// When the version value is set to "true" (--version) we will print the version information for kobs. After we
	// printed the version information the application is stopped.
	if showVersion {
		v, err := version.Print("rcloneui")
		if err != nil {
			log.Fatalf("Could not print version information: %#v", err)
		}

		fmt.Fprintln(os.Stdout, v)
		return
	}

	// Load the rclone configuration file and get a list of all sections. The sections are always used as entrypoint for
	// the rcloneui.
	configfile.LoadConfig(context.Background())
	remotes := config.Data.GetSectionList()

	// Get the users current directory, which is used as destination for downloading files.
	userDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get current directory: %#v", err)
	}

	for {
		remote, err := prompt.SelectRemote(remotes)
		if err != nil {
			if err == promptui.ErrInterrupt {
				return
			}
			log.Fatalf("Could not show select remote view: %#v", err)
		}

		var path []string

		for {
			f, err := fs.NewFs(context.Background(), fmt.Sprintf("%s:%s", remote, strings.Join(path, "/")))
			if err != nil {
				if err == fs.ErrorIsFile {
					action, err := prompt.SelectFileAction(remote, path)
					if err != nil {
						if err == promptui.ErrInterrupt {
							return
						}
						log.Fatalf("Could not show file action view: %#v", err)
					}

					if action == "Cancel" {
						path = path[:len(path)-1]
					} else if action == "Copy" {
						fdst, err := fs.NewFs(context.Background(), userDir)
						if err != nil {
							log.Fatalf("Could not create new fsrc object: %#v", err)
						}

						fsrc, err := fs.NewFs(context.Background(), fmt.Sprintf("%s:%s", remote, strings.Join(path[:len(path)-1], "/")))
						if err != nil {
							log.Fatalf("Could not create new fsrc object: %#v", err)
						}

						filename := path[len(path)-1]
						err = operations.CopyFile(context.Background(), fdst, fsrc, filename, filename)
						if err != nil {
							log.Fatalf("Could not copy file: %#v", err)
						}

						path = path[:len(path)-1]
					} else if action == "Delete" {
						fdst, err := fs.NewFs(context.Background(), fmt.Sprintf("%s:%s", remote, strings.Join(path[:len(path)-1], "/")))
						if err != nil {
							log.Fatalf("Could not create new fsrc object: %#v", err)
						}

						dst, err := fdst.NewObject(context.Background(), path[len(path)-1])
						if err != nil {
							log.Fatalf("Could not get file for deletion: %#v", err)
						}

						err = operations.DeleteFile(context.Background(), dst)
						if err != nil {
							log.Fatalf("Could not delete file: %#v", err)
						}

						path = path[:len(path)-1]
					}
				} else {
					log.Fatalf("Could not create new fs object for \"%s:%s\": %#v", remote, path, err)
				}
			}

			entries, err := f.List(context.Background(), "")
			if err != nil {
				log.Fatalf("Could not get entries for ")
			}

			convertedEntries := prompt.ConvertEntries(entries)
			entry, err := prompt.SelectEntry(remote, path, convertedEntries)
			if err != nil {
				if err == promptui.ErrInterrupt {
					return
				}
				log.Fatalf("Could not show select entry view: %#v", err)
			}

			if entry == ".." {
				if len(path) == 0 {
					break
				} else {
					path = path[:len(path)-1]
				}
			} else {
				path = append(path, entry)
			}
		}
	}
}
