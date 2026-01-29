package options

import (
	"gravity/internal/engine"
	"gravity/internal/model"
)

// Builder provides a fluent interface for constructing download options
// Deprecated: Use engine.NewOptionResolver instead.
// This file is kept for backward compatibility during migration.
type Builder struct {
	opts *engine.DownloadOptions
}

// NewBuilder starts a fresh option set with system defaults
func NewBuilder() *Builder {
	defaultSplit := 8
	defaultTimeout := 60
	defaultTries := 5
	defaultCheckCert := true
	defaultPreAllocate := false

	return &Builder{
		opts: &engine.DownloadOptions{
			Split:            &defaultSplit,
			ConnectTimeout:   &defaultTimeout,
			MaxTries:         &defaultTries,
			CheckCertificate: &defaultCheckCert,
			PreAllocateSpace: &defaultPreAllocate,
		},
	}
}

// WithSettings applies values from global settings
func (b *Builder) WithSettings(s *model.Settings) *Builder {
	if s == nil {
		return b
	}

	ds := s.Download
	us := s.Upload

	// Map settings to options
	b.opts.DownloadDir = ds.DownloadDir

	// Create local copies to avoid pointer sharing with global settings
	split := ds.Split
	b.opts.Split = &split

	connPerServer := ds.MaxConnectionPerServer
	b.opts.MaxConnectionPerServer = &connPerServer

	timeout := ds.ConnectTimeout
	b.opts.ConnectTimeout = &timeout

	tries := ds.MaxTries
	b.opts.MaxTries = &tries

	checkCert := ds.CheckCertificate
	b.opts.CheckCertificate = &checkCert

	userAgent := ds.UserAgent
	b.opts.UserAgent = &userAgent

	preAlloc := ds.PreAllocateSpace
	b.opts.PreAllocateSpace = &preAlloc

	diskCache := ds.DiskCache
	b.opts.DiskCache = &diskCache

	minSplit := ds.MinSplitSize
	b.opts.MinSplitSize = &minSplit

	maxDL := ds.MaxDownloadSpeed
	b.opts.MaxDownloadSpeed = &maxDL

	maxUL := ds.MaxUploadSpeed
	b.opts.MaxUploadSpeed = &maxUL

	lowSpeed := ds.LowestSpeedLimit
	b.opts.LowestSpeedLimit = &lowSpeed

	autoUL := us.AutoUpload
	b.opts.AutoUpload = &autoUL

	removeLocal := us.RemoveLocal
	b.opts.RemoveLocal = &removeLocal

	concurrentUL := us.ConcurrentUploads
	b.opts.ConcurrentUploads = &concurrentUL

	// Map Proxies
	if len(s.Network.Proxies) > 0 {
		var engineProxies []engine.Proxy
		for _, p := range s.Network.Proxies {
			engineProxies = append(engineProxies, engine.Proxy{
				URL:  p.URL,
				Type: p.Type,
			})
		}
		b.opts.Proxies = engineProxies
	}

	return b
}

// WithModel applies options from an existing database model (for resume/retry/request)
// This is used for both stored downloads and temporary "carrier" models from API requests.
func (b *Builder) WithModel(d *model.Download) *Builder {
	if d == nil {
		return b
	}

	if d.ID != "" {
		b.opts.ID = d.ID
	}
	if d.URL != "" {
		b.opts.URL = d.URL
	}
	if d.Filename != "" {
		b.opts.Filename = d.Filename
	}
	if d.Dir != "" {
		b.opts.DownloadDir = d.Dir
	}
	if d.Destination != "" {
		b.opts.Destination = d.Destination
	}
	if d.Split != nil {
		b.opts.Split = d.Split
	}
	if d.RemoveLocal != nil {
		b.opts.RemoveLocal = d.RemoveLocal
	}
	if d.Headers != nil {
		if b.opts.Headers == nil {
			b.opts.Headers = make(map[string]string)
		}
		for k, v := range d.Headers {
			b.opts.Headers[k] = v
		}
	}
	if d.TorrentData != "" {
		b.opts.TorrentData = d.TorrentData
	}
	if d.MagnetHash != "" {
		b.opts.MagnetHash = d.MagnetHash
	}
	if d.Engine != "" {
		b.opts.Engine = d.Engine
	}

	if len(d.SelectedFiles) > 0 {
		b.opts.SelectedFiles = d.SelectedFiles
	}

	if len(d.Proxies) > 0 {
		var engineProxies []engine.Proxy
		for _, p := range d.Proxies {
			engineProxies = append(engineProxies, engine.Proxy{
				URL:  p.URL,
				Type: p.Type,
			})
		}
		b.opts.Proxies = engineProxies
	}

	return b
}

// Build returns the final options
func (b *Builder) Build() *engine.DownloadOptions {
	return b.opts
}
