package service

import (
	"gravity/internal/engine"
	"gravity/internal/model"
)

// TaskOptions helpers for converting between model and engine types

// toEngineOptions converts model.TaskOptions to engine.DownloadOptions
func toEngineOptions(opts model.TaskOptions) engine.DownloadOptions {
	return engine.DownloadOptions{
		DownloadDir: opts.DownloadDir,
		Destination: opts.Destination,
		Split:       intPtr(opts.Split),
		MaxTries:    intPtr(opts.MaxTries),
		UserAgent:   strPtr(opts.UserAgent),
		ProxyURL:    strPtr(opts.ProxyURL),
		RemoveLocal: opts.RemoveLocal,
		Headers:     opts.Headers,
	}
}

// toEngineOptionsFromDownload converts a Download's stored options to engine.DownloadOptions
func toEngineOptionsFromDownload(d *model.Download) engine.DownloadOptions {
	opts := toEngineOptions(d.Options)

	// Use top-level fields if Options doesn't have them set
	if opts.DownloadDir == "" {
		opts.DownloadDir = d.DownloadDir
	}
	if opts.Destination == "" {
		opts.Destination = d.Destination
	}

	return opts
}

// fromEngineOptions converts engine.DownloadOptions back to model.TaskOptions
func fromEngineOptions(opts engine.DownloadOptions) model.TaskOptions {
	return model.TaskOptions{
		DownloadDir: opts.DownloadDir,
		Destination: opts.Destination,
		Split:       derefInt(opts.Split),
		MaxTries:    derefInt(opts.MaxTries),
		UserAgent:   derefString(opts.UserAgent),
		ProxyURL:    derefString(opts.ProxyURL),
		RemoveLocal: opts.RemoveLocal,
		Headers:     opts.Headers,
	}
}

// Helper functions

func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
