package main

import (
	"fmt"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// LargerTheme extends the default theme with larger font sizes
type LargerTheme struct {
	fyne.Theme
}

// Font returns a larger font resource
func (t *LargerTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 16 // Increase from default 14
	case theme.SizeNameHeadingText:
		return 24 // Increase from default 20
	case theme.SizeNameSubHeadingText:
		return 18 // Increase from default 16
	case theme.SizeNameInputBorder:
		return 2 // Slightly larger borders
	case theme.SizeNamePadding:
		return 6 // Slightly more padding
	case theme.SizeNameScrollBar:
		return 16 // Larger scrollbar
	case theme.SizeNameScrollBarSmall:
		return 8 // Larger small scrollbar
	default:
		return t.Theme.Size(name)
	}
}

// TorrentResult represents a single search result
type TorrentResult struct {
	Name          string
	Size          string
	Seeders       int
	Leechers      int
	DateAdded     string
	MagnetLink    string
	IsDownloading bool
	IsDownloaded  bool
}

// MockData generates fake torrent results based on search query
func generateMockResults(query string) []TorrentResult {
	if query == "" {
		return []TorrentResult{}
	}

	// Seed random generator
	rand.Seed(time.Now().UnixNano())

	// Random number of results (3-10)
	numResults := rand.Intn(8) + 3

	categories := []string{"Movies", "TV Shows", "Games", "Software", "Music", "Books"}
	qualities := []string{"HD", "4K", "720p", "1080p", "FLAC", "MP3", "PDF"}

	results := make([]TorrentResult, numResults)

	for i := 0; i < numResults; i++ {
		category := categories[rand.Intn(len(categories))]
		quality := qualities[rand.Intn(len(qualities))]

		// Size between 100MB and 20GB
		sizeInMB := rand.Intn(20000) + 100
		sizeStr := ""
		if sizeInMB >= 1000 {
			sizeStr = fmt.Sprintf("%.1f GB", float64(sizeInMB)/1000)
		} else {
			sizeStr = fmt.Sprintf("%d MB", sizeInMB)
		}

		// Date within last 60 days
		daysAgo := rand.Intn(60)
		date := time.Now().AddDate(0, 0, -daysAgo)
		dateStr := date.Format("Jan 02, 2006")

		// Generate name based on query and random elements
		name := fmt.Sprintf("%s - %s [%s]", query, category, quality)

		results[i] = TorrentResult{
			Name:          name,
			Size:          sizeStr,
			Seeders:       rand.Intn(500) + 1,
			Leechers:      rand.Intn(200),
			DateAdded:     dateStr,
			MagnetLink:    fmt.Sprintf("magnet:?xt=urn:btih:%x", rand.Int63()),
			IsDownloading: false,
			IsDownloaded:  false,
		}
	}

	return results
}

func main() {
	// Initialize app
	myApp := app.New()

	// Set custom theme with larger font
	myApp.Settings().SetTheme(&LargerTheme{Theme: theme.DefaultTheme()})

	myWindow := myApp.NewWindow("Cactus app")
	myWindow.Resize(fyne.NewSize(1000, 700)) // Larger default window size

	// Storage for current results
	var currentResults []TorrentResult

	// Create status bar at the bottom - now with larger text
	statusLabel := widget.NewLabel("Ready")
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	// First, make a template for list items
	listItemTemplate := func() fyne.CanvasObject {
		// Name with truncation
		nameLabel := widget.NewLabel("Name placeholder text")
		nameLabel.Wrapping = fyne.TextWrapWord
		nameLabel.TextStyle = fyne.TextStyle{Bold: true}

		// Details in smaller text but still readable
		detailsLabel := widget.NewLabel("Details placeholder")
		detailsLabel.TextStyle = fyne.TextStyle{Italic: true}

		// Create a wider download button
		downloadButton := widget.NewButtonWithIcon("Download", theme.DownloadIcon(), nil)
		downloadButton.Importance = widget.HighImportance // Make button more prominent

		// Set minimum width for the button - make it approximately twice as wide
		downloadButtonContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(160, 40)), downloadButton)

		// Layout with button on right and more padding
		content := container.NewVBox(
			nameLabel,
			detailsLabel,
		)

		// Add padding around content
		paddedContent := container.NewPadded(content)

		return container.NewBorder(
			nil, nil, nil, downloadButtonContainer,
			paddedContent,
		)
	}

	// Now create the list with the template
	resultList := widget.NewList(
		// Length function
		func() int {
			return len(currentResults)
		},
		// Create template for each item
		listItemTemplate,
		// Update function comes later with a separate function
		nil,
	)

	// Make list items taller for better readability
	//resultList.SetItemHeight(1, 105)

	// Define update function separately to avoid self-reference
	updateFunc := func(id widget.ListItemID, item fyne.CanvasObject) {
		// Skip if index invalid
		if id >= len(currentResults) {
			return
		}

		// Get result and container
		result := currentResults[id]
		itemContainer := item.(*fyne.Container)

		// Get child widgets - need to navigate through the container hierarchy
		paddedContent := itemContainer.Objects[0].(*fyne.Container)
		contentBox := paddedContent.Objects[0].(*fyne.Container)
		nameLabel := contentBox.Objects[0].(*widget.Label)
		detailsLabel := contentBox.Objects[1].(*widget.Label)

		// Get the button container and then the button within it
		buttonContainer := itemContainer.Objects[1].(*fyne.Container)
		downloadButton := buttonContainer.Objects[0].(*widget.Button)

		// Update content
		nameLabel.SetText(result.Name)
		detailsLabel.SetText(fmt.Sprintf("Size: %s | Seeds: %d | Peers: %d | Added: %s",
			result.Size, result.Seeders, result.Leechers, result.DateAdded))

		// Update button state and text based on download status
		if result.IsDownloaded {
			downloadButton.SetText("Downloaded")
			downloadButton.SetIcon(theme.ConfirmIcon())
			downloadButton.Disable()
		} else if result.IsDownloading {
			downloadButton.SetText("Downloading...")
			downloadButton.SetIcon(theme.DownloadIcon())
			downloadButton.Disable()
		} else {
			downloadButton.SetText("Download")
			downloadButton.SetIcon(theme.DownloadIcon())
			downloadButton.Enable()
		}

		// Set button action
		downloadButton.OnTapped = func() {
			// Mark as downloading
			currentResults[id].IsDownloading = true
			resultList.Refresh()

			// Update status
			statusLabel.SetText(fmt.Sprintf("Downloading: %s", result.Name))

			// Simulate download process with goroutine
			go func() {
				// Show download dialog
				dialog.ShowInformation(
					"Download Started",
					fmt.Sprintf("Starting download for: %s\n\nMagnet link: %s",
						result.Name, result.MagnetLink),
					myWindow,
				)

				// Simulate download completion after delay
				time.Sleep(3 * time.Second)

				// Use the safe approach to update UI from a goroutine in Fyne
				myApp.Driver().DoFromGoroutine(func() {
					currentResults[id].IsDownloading = false
					currentResults[id].IsDownloaded = true
					resultList.Refresh()
					statusLabel.SetText("Download complete: " + result.Name)
				}, false)
			}()
		}
	}

	// Now set the update function
	resultList.UpdateItem = updateFunc

	// Create search interface
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Enter search terms...")
	searchEntry.TextStyle = fyne.TextStyle{Bold: true}

	// Search button with action - larger and more prominent
	searchButton := widget.NewButtonWithIcon("Search", theme.SearchIcon(), func() {
		query := searchEntry.Text
		if query == "" {
			dialog.ShowInformation("Error", "Please enter a search term", myWindow)
			return
		}

		// Update status
		statusLabel.SetText("Searching for: " + query)

		// Simulate search delay
		go func() {
			time.Sleep(1 * time.Second)

			// Generate results in the goroutine
			mockResults := generateMockResults(query)

			// Update UI on the main thread using the correct Fyne API
			myApp.Driver().DoFromGoroutine(func() {
				currentResults = mockResults
				resultList.Refresh()

				// Update status
				if len(currentResults) > 0 {
					statusLabel.SetText(fmt.Sprintf("Found %d results for '%s'", len(currentResults), query))
				} else {
					statusLabel.SetText(fmt.Sprintf("No results found for '%s'", query))
				}
			}, false)
		}()
	})
	searchButton.Importance = widget.HighImportance

	// Allow Enter key to search
	searchEntry.OnSubmitted = func(s string) {
		searchButton.OnTapped()
	}

	// Create search container with padding
	searchContainer := container.NewPadded(
		container.NewBorder(
			nil, nil, nil, searchButton,
			searchEntry,
		),
	)

	// Create the status bar with more visibility
	statusBar := container.NewPadded(
		container.NewHBox(statusLabel),
	)

	// Create tab panels with larger tab text
	searchTab := container.NewBorder(
		searchContainer,
		statusBar,
		nil, nil,
		container.NewPadded(resultList),
	)

	// Create other tabs with larger, centered text
	shareLabel := widget.NewLabel("Share functionality will be implemented in a future version")
	shareLabel.TextStyle = fyne.TextStyle{Bold: true}

	controlLabel := widget.NewLabel("Control Panel will be implemented in a future version")
	controlLabel.TextStyle = fyne.TextStyle{Bold: true}

	shareTab := container.NewCenter(shareLabel)
	controlTab := container.NewCenter(controlLabel)

	// Create tabs container
	tabs := container.NewAppTabs(
		container.NewTabItem("Search", searchTab),
		container.NewTabItem("Share", shareTab),
		container.NewTabItem("Control Panel", controlTab),
	)

	// Make tabs more prominent
	tabs.SetTabLocation(container.TabLocationTop)

	// Set window content and show
	myWindow.SetContent(tabs)
	myWindow.ShowAndRun()
}
