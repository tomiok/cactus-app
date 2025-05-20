package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/anacrolix/log"
	"github.com/tomiok/cactus-app/downloader"
	"io"
	"net/http"
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

type Result struct {
	FirstSeen    string    `json:"FirstSeen"`
	Tracker      string    `json:"Tracker"`
	TrackerId    string    `json:"TrackerId"`
	TrackerType  string    `json:"TrackerType"`
	CategoryDesc string    `json:"CategoryDesc"`
	Title        string    `json:"Title"`
	Guid         string    `json:"Guid"`
	Link         string    `json:"Link"`
	Details      string    `json:"Details"`
	Peers        int       `json:"Peers"`
	Seeders      int       `json:"Seeders"`
	PublishDate  time.Time `json:"PublishDate"`
	Size         int64     `json:"Size"`
	Categories   []int     `json:"Category"`
	MagnetUri    string    `json:"MagnetUri"`

	IsDownloading bool `json:"-"`
	IsDownloaded  bool `json:"-"`
}

// SearchResults is the structure for the API response
type SearchResults struct {
	Results []Result `json:"results"`
	Total   int      `json:"total"`
	Query   string   `json:"query"`
}

type SearchRequest struct {
	Query      string   `json:"query"`
	Categories []string `json:"categories"`
	Trackers   []string `json:"trackers"`
}

const apiURL = "http://localhost:7000/search"

func fetchSearchResults(query string) (SearchResults, error) {
	req := SearchRequest{Query: query}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return SearchResults{}, err
	}
	buff := bytes.NewBuffer(body)
	// Make the request
	resp, err := client.Post(apiURL, "application/json", buff)
	if err != nil || resp.StatusCode != http.StatusOK {
		return SearchResults{}, fmt.Errorf("API request failed: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var searchResults SearchResults

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResults{}, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if err = json.Unmarshal(b, &searchResults); err != nil {
		return SearchResults{}, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	var curatedList SearchResults
	for _, res := range searchResults.Results {
		if res.MagnetUri != "" {
			curatedList.Results = append(curatedList.Results, res)
		}
	}

	curatedList.Total = len(curatedList.Results)
	return curatedList, nil
}

func main() {
	myApp := app.New()

	myApp.Settings().SetTheme(&LargerTheme{Theme: theme.DefaultTheme()})

	myWindow := myApp.NewWindow("Cactus app")
	myWindow.Resize(fyne.NewSize(1000, 700)) // Larger default window size

	var searchResult SearchResults

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
			return searchResult.Total
		},
		// Create template for each item
		listItemTemplate,
		// Update function comes later with a separate function
		nil,
	)

	// Define update function separately to avoid self-reference
	updateFunc := func(id widget.ListItemID, item fyne.CanvasObject) {
		// Skip if index invalid
		if id >= len(searchResult.Results) {
			return
		}

		// Get result and container
		result := searchResult.Results[id]
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
		nameLabel.SetText(result.Title)
		detailsLabel.SetText(fmt.Sprintf("Size: %d | Seeds: %d | Peers: %d | Link: %s",
			result.Size, result.Seeders, result.Peers, result.MagnetUri))

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
			searchResult.Results[id].IsDownloading = true
			resultList.Refresh()

			// Update status
			statusLabel.SetText(fmt.Sprintf("Downloading: %s", result.Title))

			// Simulate download process with goroutine
			go func() {
				// Show download dialog
				dialog.ShowInformation(
					"Download Started",
					fmt.Sprintf("Starting download for: %s\n\nMagnet link: %s",
						result.Title, result.MagnetUri),
					myWindow,
				)

				td, err := downloader.NewTorrentDownloader("~/Downloads")
				if err != nil {
					log.Printf("cannot create directory %s \n", err)
					return
				}

				title, err := td.DownloadFromMagnet(context.Background(), result.MagnetUri, func(info downloader.ProgressInfo) {
					// Format download speed
					var speedStr string
					if info.DownloadSpeed < 1024 {
						speedStr = fmt.Sprintf("%.2f B/s", info.DownloadSpeed)
					} else if info.DownloadSpeed < 1024*1024 {
						speedStr = fmt.Sprintf("%.2f KB/s", info.DownloadSpeed/1024)
					} else {
						speedStr = fmt.Sprintf("%.2f MB/s", info.DownloadSpeed/(1024*1024))
					}

					fmt.Printf("\rProgress: %.2f%% (%.2f MB/%.2f MB) - Peers: %d - Speed: %s",
						info.PercentDone,
						float64(info.BytesCompleted)/(1024*1024),
						float64(info.TotalBytes)/(1024*1024),
						info.PeersConnected,
						speedStr,
					)
				})

				if err != nil {
					log.Printf("cannot complete download %s \n", err)
					return
				}

				// Use the safe approach to update UI from a goroutine in Fyne
				myApp.Driver().DoFromGoroutine(func() {
					searchResult.Results[id].IsDownloading = false
					searchResult.Results[id].IsDownloaded = true
					resultList.Refresh()
					statusLabel.SetText("Download complete: " + title)
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

	// Error dialog helper function
	showError := func(title, message string) {
		dialog.ShowError(fmt.Errorf("%s", message), myWindow)
	}

	// Search button with action - larger and more prominent
	searchButton := widget.NewButtonWithIcon("Search", theme.SearchIcon(), func() {
		query := searchEntry.Text
		if query == "" {
			dialog.ShowInformation("Error", "Please enter a search term", myWindow)
			return
		}

		// Update status
		statusLabel.SetText("Searching for: " + query)

		// Perform the API search in a goroutine
		go func() {
			// Call the API
			result, err := fetchSearchResults(query)

			// Update UI on the main thread
			myApp.Driver().DoFromGoroutine(func() {
				if err != nil {
					statusLabel.SetText("Search failed")
					showError("Search Error", fmt.Sprintf("Failed to search: %v", err))
					return
				}

				// Update the results
				searchResult = result
				resultList.Refresh()

				// Update status based on results
				if result.Total > 0 {
					statusLabel.SetText(fmt.Sprintf("Found %d results for '%s'", result.Total, query))
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
