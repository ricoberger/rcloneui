package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ricoberger/rcloneui/pkg/version"
	"github.com/ricoberger/rcloneui/pkg/view"

	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/rivo/tview"
	flag "github.com/spf13/pflag"
)

var (
	maxAge      string
	maxSize     string
	minAge      string
	minSize     string
	showVersion bool
)

// init is used to define all flags for rcloneui. For example we define the --version flag here, which can be used to
// print the version information of rcloneui.
func init() {
	flag.StringVar(&maxAge, "max-age", "off", "Only transfer files younger than this in s or suffix ms|s|m|h|d|w|M|y.")
	flag.StringVar(&maxSize, "max-size", "off", "Only transfer files smaller than this in k or suffix b|k|M|G.")
	flag.StringVar(&minAge, "min-age", "off", "Only transfer files older than this in s or suffix ms|s|m|h|d|w|M|y.")
	flag.StringVar(&minSize, "min-size", "off", "Only transfer files bigger than this in k or suffix b|k|M|G.")
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
	remotes = append([]string{view.Local}, remotes...)

	// Get the users current directory, which is used as destination for downloading files.
	userDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get current directory: %#v", err)
	}

	// Initialize the status bar, the two views and the grid, which then are rendered via tview. After the views are
	// initialized we have to pass the other view to a view, so that we can switch the focus via the tab key.
	app := tview.NewApplication()

	filter, err := view.CreateFilter(minAge, maxAge, minSize, maxSize)
	if err != nil {
		log.Fatalf("Could not create filter: %#v", err)
	}

	status := view.NewStatus(app)
	view1 := view.NewView(app, status, remotes, strings.Split(userDir, "/"), filter)
	view2 := view.NewView(app, status, remotes, strings.Split(userDir, "/"), filter)

	view1.SetView(view2)
	view2.SetView(view1)

	grid := tview.NewGrid().SetRows(0, 1).SetColumns(0, 0).SetBorders(true)
	grid.SetBordersColor(tcell.ColorBlack)
	grid.AddItem(view1, 0, 0, 1, 1, 0, 0, true).AddItem(view2, 0, 1, 1, 1, 0, 0, false)
	grid.AddItem(status, 1, 0, 1, 2, 0, 0, false)

	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		log.Fatalf("Could not render view: %#v", err)
	}
}
