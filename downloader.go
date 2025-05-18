package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
)

// ProgressInfo contains information about download progress
type ProgressInfo struct {
	PercentDone    float64
	BytesCompleted int64
	TotalBytes     int64
	PeersConnected int
	DownloadSpeed  float64 // in bytes per second
}

// TorrentDownloader manages torrent downloads
type TorrentDownloader struct {
	client       *torrent.Client
	downloadPath string
}

// NewTorrentDownloader creates a new downloader instance
func NewTorrentDownloader(downloadPath string) (*TorrentDownloader, error) {
	// Create download directory if it doesn't exist
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	// Create a torrent client with disk-based storage
	cfg := torrent.NewDefaultClientConfig()
	cfg.DefaultStorage = storage.NewFile(downloadPath)
	cfg.DataDir = downloadPath

	// Set reasonable limits to prevent system overload
	cfg.EstablishedConnsPerTorrent = 31
	cfg.Seed = false       // Set to true if you want to seed after download
	cfg.DisableIPv6 = true // Disable IPv6 if it's causing issues
	cfg.DisableTCP = false // Keep TCP enabled for better connectivity
	cfg.DisableUTP = false // Keep uTP enabled for better NAT traversal
	cfg.NoDHT = false      // Keep DHT enabled for better peer discovery
	cfg.NoUpload = true    // Disable uploading to improve download performance
	cfg.ListenPort = 0

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent client: %w", err)
	}

	return &TorrentDownloader{
		client:       client,
		downloadPath: downloadPath,
	}, nil
}

// Close shuts down the torrent client
func (td *TorrentDownloader) Close() error {
	td.client.Close()
	return nil
}

// DownloadFromMagnet downloads a file from a magnet link
func (td *TorrentDownloader) DownloadFromMagnet(
	ctx context.Context,
	magnetLink string,
	progressCallback func(ProgressInfo),
) (string, error) {
	// Add the torrent from the magnet link
	t, err := td.client.AddMagnet(magnetLink)
	if err != nil {
		return "", fmt.Errorf("failed to add magnet: %w", err)
	}

	// Wait for torrent metadata with context
	select {
	case <-t.GotInfo():
		// Got the info
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(2 * time.Minute):
		return "", errors.New("timeout waiting for torrent metadata")
	}

	// Start downloading
	t.DownloadAll()

	// Create a ticker for progress updates
	progressTicker := time.NewTicker(1 * time.Second)
	defer progressTicker.Stop()

	var lastBytes int64
	var lastTime = time.Now()

	// Monitor download progress
	for {
		select {
		case <-progressTicker.C:
			currentTime := time.Now()
			bytesCompleted := t.BytesCompleted()
			totalBytes := t.Length()
			percentDone := float64(bytesCompleted) / float64(totalBytes) * 100

			// Calculate download speed
			timeElapsed := currentTime.Sub(lastTime).Seconds()
			bytesDownloaded := bytesCompleted - lastBytes
			downloadSpeed := float64(bytesDownloaded) / timeElapsed

			lastBytes = bytesCompleted
			lastTime = currentTime

			if progressCallback != nil {
				progressCallback(ProgressInfo{
					PercentDone:    percentDone,
					BytesCompleted: bytesCompleted,
					TotalBytes:     totalBytes,
					PeersConnected: len(t.PeerConns()),
					DownloadSpeed:  downloadSpeed,
				})
			}

			// Check if download is complete
			if t.Complete().Bool() {
				// Get the path to the downloaded file
				info := t.Info()
				if len(info.Files) == 0 {
					// Single file torrent
					filePath := filepath.Join(td.downloadPath, info.Name)
					return filePath, nil
				}
				// Return the path to the directory for multi-file torrents
				return filepath.Join(td.downloadPath, info.Name), nil
			}

		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func main() {
	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling to cancel downloads when CTRL+C is pressed
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		fmt.Println("\nCancelling download...")
		cancel()
	}()

	// Create a downloader
	downloader, err := NewTorrentDownloader(".")
	if err != nil {
		fmt.Printf("Error creating downloader: %v\n", err)
		return
	}
	defer downloader.Close()

	// The magnet link to download
	magnetLink := "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=ubuntu-22.04-desktop-amd64.iso"

	fmt.Println("Starting download...")

	// Start the download
	filePath, err := downloader.DownloadFromMagnet(ctx, magnetLink, func(info ProgressInfo) {
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
		fmt.Printf("\nDownload error: %v\n", err)
		return
	}

	fmt.Printf("\nDownload complete! File saved to: %s\n", filePath)
}
