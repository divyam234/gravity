package store

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

const (
	BucketUploads  = "uploads"
	BucketStats    = "stats"
	DbName         = "aria2-rclone.db"
	KeyGlobalStats = "global"
)

type UploadState struct {
	Gid          string                 `json:"gid"`
	TargetRemote string                 `json:"targetRemote"`    // e.g. "gdrive:"
	JobID        string                 `json:"jobId,omitempty"` // Rclone Job ID (string for JSON compat)
	Status       string                 `json:"status"`          // pending, uploading, complete, error
	StartedAt    time.Time              `json:"startedAt"`
	LastError    string                 `json:"lastError,omitempty"`
	URIs         []string               `json:"uris,omitempty"`
	Torrent      string                 `json:"torrent,omitempty"`  // Base64 encoded
	Metalink     string                 `json:"metalink,omitempty"` // Base64 encoded
	Options      map[string]interface{} `json:"options,omitempty"`
	RetryCount   int                    `json:"retryCount,omitempty"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	// Stats for this task
	DownloadedBytes int64 `json:"downloadedBytes,omitempty"`
	UploadedBytes   int64 `json:"uploadedBytes,omitempty"`
}

// GlobalStats stores cumulative transfer statistics
type GlobalStats struct {
	TotalDownloaded int64     `json:"totalDownloaded"` // Total bytes downloaded (all time)
	TotalUploaded   int64     `json:"totalUploaded"`   // Total bytes uploaded to cloud (all time)
	TotalTasks      int64     `json:"totalTasks"`      // Total tasks ever created
	CompletedTasks  int64     `json:"completedTasks"`  // Tasks that finished download
	UploadedTasks   int64     `json:"uploadedTasks"`   // Tasks that finished upload to cloud
	LastUpdated     time.Time `json:"lastUpdated"`
}

type DB struct {
	db *bbolt.DB
}

func New(path string) (*DB, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(BucketUploads)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(BucketStats)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) SaveUpload(state UploadState) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(state.Gid), data)
	})
}

func (d *DB) GetUpload(gid string) (*UploadState, error) {
	var state UploadState
	err := d.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		data := b.Get([]byte(gid))
		if data == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(data, &state)
	})
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (d *DB) GetPendingUploads() ([]UploadState, error) {
	var uploads []UploadState
	err := d.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		return b.ForEach(func(k, v []byte) error {
			var state UploadState
			if err := json.Unmarshal(v, &state); err != nil {
				return nil // Skip malformed
			}
			// We want to restore tasks that are not complete/uploaded
			if state.Status != "complete" && state.Status != "error" && state.Status != "removed" {
				uploads = append(uploads, state)
			}
			return nil
		})
	})
	return uploads, err
}

func (d *DB) ResetStuckUploads() error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		return b.ForEach(func(k, v []byte) error {
			var state UploadState
			if err := json.Unmarshal(v, &state); err != nil {
				return nil
			}
			if state.Status == "uploading" {
				state.Status = "pending"
				state.UpdatedAt = time.Now()
				data, _ := json.Marshal(state)
				return b.Put(k, data)
			}
			return nil
		})
	})
}

func (d *DB) UpdateStatus(gid string, status string, jobId string) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		data := b.Get([]byte(gid))
		if data == nil {
			return fmt.Errorf("upload not found for gid %s", gid)
		}

		var state UploadState
		if err := json.Unmarshal(data, &state); err != nil {
			return err
		}

		state.Status = status
		state.UpdatedAt = time.Now()
		if jobId != "" {
			state.JobID = jobId
		}

		newData, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(gid), newData)
	})
}

func (d *DB) UpdateJob(gid string, jobId string) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		data := b.Get([]byte(gid))
		if data == nil {
			return fmt.Errorf("not found")
		}

		var state UploadState
		if err := json.Unmarshal(data, &state); err != nil {
			return err
		}

		state.JobID = jobId
		state.UpdatedAt = time.Now()

		newData, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(gid), newData)
	})
}

func (d *DB) IncrementRetry(gid string) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		data := b.Get([]byte(gid))
		if data == nil {
			return fmt.Errorf("not found")
		}

		var state UploadState
		if err := json.Unmarshal(data, &state); err != nil {
			return err
		}

		state.RetryCount++
		state.UpdatedAt = time.Now()

		newData, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(gid), newData)
	})
}

func (d *DB) UpdateOptions(gid string, options map[string]interface{}) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		data := b.Get([]byte(gid))
		if data == nil {
			return fmt.Errorf("not found")
		}

		var state UploadState
		if err := json.Unmarshal(data, &state); err != nil {
			return err
		}

		// Merge options
		if state.Options == nil {
			state.Options = make(map[string]interface{})
		}
		for k, v := range options {
			state.Options[k] = v
		}
		state.UpdatedAt = time.Now()

		newData, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(gid), newData)
	})
}

// GetGlobalStats returns cumulative transfer statistics
func (d *DB) GetGlobalStats() (*GlobalStats, error) {
	var stats GlobalStats
	err := d.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketStats))
		data := b.Get([]byte(KeyGlobalStats))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &stats)
	})
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// UpdateGlobalStats atomically updates cumulative statistics
func (d *DB) UpdateGlobalStats(fn func(*GlobalStats)) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketStats))
		var stats GlobalStats

		data := b.Get([]byte(KeyGlobalStats))
		if data != nil {
			if err := json.Unmarshal(data, &stats); err != nil {
				return err
			}
		}

		fn(&stats)
		stats.LastUpdated = time.Now()

		newData, err := json.Marshal(stats)
		if err != nil {
			return err
		}
		return b.Put([]byte(KeyGlobalStats), newData)
	})
}

// AddDownloadedBytes increments total downloaded bytes
func (d *DB) AddDownloadedBytes(bytes int64) error {
	return d.UpdateGlobalStats(func(s *GlobalStats) {
		s.TotalDownloaded += bytes
	})
}

// AddUploadedBytes increments total uploaded bytes
func (d *DB) AddUploadedBytes(bytes int64) error {
	return d.UpdateGlobalStats(func(s *GlobalStats) {
		s.TotalUploaded += bytes
	})
}

// IncrementCompletedTasks increments completed download count
func (d *DB) IncrementCompletedTasks() error {
	return d.UpdateGlobalStats(func(s *GlobalStats) {
		s.CompletedTasks++
	})
}

// IncrementUploadedTasks increments completed upload count
func (d *DB) IncrementUploadedTasks() error {
	return d.UpdateGlobalStats(func(s *GlobalStats) {
		s.UploadedTasks++
	})
}

// IncrementTotalTasks increments total task count
func (d *DB) IncrementTotalTasks() error {
	return d.UpdateGlobalStats(func(s *GlobalStats) {
		s.TotalTasks++
	})
}

// GetUploadingCount returns number of currently uploading tasks
func (d *DB) GetUploadingCount() (int, error) {
	count := 0
	err := d.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		return b.ForEach(func(k, v []byte) error {
			var state UploadState
			if err := json.Unmarshal(v, &state); err != nil {
				return nil
			}
			if state.Status == "uploading" {
				count++
			}
			return nil
		})
	})
	return count, err
}
