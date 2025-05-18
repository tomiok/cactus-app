package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
)

func Test_Download(t *testing.T) {
	magnetLink := "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=ubuntu-22.04-desktop-amd64.iso"

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Optional: Handle Ctrl+C to cancel the download
	go func() {
		c := make(chan os.Signal, 1)
		// signal.Notify(c, os.Interrupt)
		<-c
		fmt.Println("\n cancelling download...")
		cancel()
	}()

	// Define a progress callback
	progressCb := func(p ProgressInfo) {
		fmt.Printf("\rProgress: %.2f%% (%d/%d bytes) - from %d peers", p.PercentDone, p.BytesCompleted, p.TotalBytes, p.PeersConnected)
	}

	fmt.Println("Starting download from magnet link...")
	dt, _ := NewTorrentDownloader(".")
	fileName, err := dt.DownloadFromMagnet(ctx, magnetLink, progressCb)
	if err != nil {
		slog.Error("Error downloading from magnet:", "err", err)
	}

	fmt.Println("downloaded:" + fileName)

	fmt.Println("File saved to 'downloaded_file'")
}
