package service

import (
	"gravity/internal/engine"
	"gravity/internal/model"
)

// TaskOptions helpers for converting between model and engine types

// toEngineOptionsFromDownload converts a Download's stored options to engine.DownloadOptions
func toEngineOptionsFromDownload(d *model.Download) engine.DownloadOptions {
	return engine.DownloadOptions{
		DownloadDir: d.DownloadDir,
		Destination: d.Destination,
		Split:       d.Split,
		MaxTries:    d.MaxTries,
		UserAgent:   d.UserAgent,
		ProxyURL:    d.ProxyURL,
		RemoveLocal: d.RemoveLocal,
		Headers:     d.Headers,
	}
}

// fromEngineOptions converts engine.DownloadOptions back to model.Download carrier
// Note: This now returns a partial model.Download-like struct since TaskOptions is gone.
// We only use this for reconstructing options, so we can return the individual fields or a struct.
// For now, let's just return what we need or remove it if unused.
// Checking usages... it was used in tests mainly.
// Let's keep a helper that returns model.Download with options set.
func fromEngineOptions(opts engine.DownloadOptions) model.Download {
	return model.Download{
		DownloadDir: opts.DownloadDir,
		Destination: opts.Destination,
		Split:       opts.Split,
		MaxTries:    opts.MaxTries,
		UserAgent:   opts.UserAgent,
		ProxyURL:    opts.ProxyURL,
		RemoveLocal: opts.RemoveLocal,
		Headers:     opts.Headers,
	}
}

// Helper functions

