package store

import (
	"encoding/json"
	"fmt"
	"sort"
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
	// Metadata for tellUploading
	FilePath    string `json:"filePath,omitempty"`
	TotalLength int64  `json:"totalLength,omitempty"`
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

// UpdateProgress updates only the progress fields of a task
func (d *DB) UpdateProgress(gid string, downloadedBytes, totalLength int64) error {
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

		state.DownloadedBytes = downloadedBytes
		state.TotalLength = totalLength
		state.UpdatedAt = time.Now()

		newData, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(gid), newData)
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

// UpdateError marks a task as error and saves the error message
func (d *DB) UpdateError(gid string, errorMessage string) error {
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

		state.Status = "error"
		state.LastError = errorMessage
		state.UpdatedAt = time.Now()

		newData, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(gid), newData)
	})
}

// UpsertTask creates or updates a task (used for startup sync and shadow import)
func (d *DB) UpsertTask(state UploadState) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))

		// If it exists, preserve some fields like Rclone target if not provided in new state
		existingData := b.Get([]byte(state.Gid))
		if existingData != nil {
			var existing UploadState
			if err := json.Unmarshal(existingData, &existing); err == nil {
				// Preserve TargetRemote if the incoming state is just from Aria2 (which doesn't know about Remotes)
				if state.TargetRemote == "" {
					state.TargetRemote = existing.TargetRemote
				}
				// Preserve Options
				if state.Options == nil {
					state.Options = existing.Options
				}
				// Preserve RetryCount
				state.RetryCount = existing.RetryCount

				// Keep JobID if we are just updating Aria2 status
				if state.JobID == "" {
					state.JobID = existing.JobID
				}

				// If the existing state is "uploading" or "complete", we should be careful about overwriting it
				// with "active" from Aria2 unless we are sure.
				// But for now, we assume Aria2 is the truth for download status.
			}
		}

		state.UpdatedAt = time.Now()
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(state.Gid), data)
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

// StartUpload marks a task as uploading and stores metadata for tellUploading.
// Returns error if the task is already uploading or finished.
func (d *DB) StartUpload(gid, jobId, filePath string, totalLength int64) error {
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

		// Prevent race: Only start if it's currently pending
		if state.Status != "pending" {
			return fmt.Errorf("task %s is already in state %s", gid, state.Status)
		}

		state.Status = "uploading"
		state.JobID = jobId
		state.FilePath = filePath
		state.TotalLength = totalLength
		state.UpdatedAt = time.Now()

		newData, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(gid), newData)
	})
}

// GetUploadingTasks returns all tasks currently in uploading state, sorted by updated time descending
func (d *DB) GetUploadingTasks() ([]UploadState, error) {
	var uploads []UploadState
	err := d.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		return b.ForEach(func(k, v []byte) error {
			var state UploadState
			if err := json.Unmarshal(v, &state); err != nil {
				return nil
			}
			if state.Status == "uploading" {
				uploads = append(uploads, state)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// Sort by UpdatedAt descending
	sort.Slice(uploads, func(i, j int) bool {
		return uploads[i].UpdatedAt.After(uploads[j].UpdatedAt)
	})

	return uploads, nil
}

// AdoptChildTask creates a new tracking record for a derived task (e.g. magnet handoff)
func (d *DB) AdoptChildTask(childGid, parentGid string, targetRemote string, options map[string]interface{}) (bool, error) {
	// Check if we already track it
	_, err := d.GetUpload(childGid)
	if err == nil {
		return false, nil // Already tracked
	}

	err = d.SaveUpload(UploadState{
		Gid:          childGid,
		TargetRemote: targetRemote,
		Status:       "pending",
		StartedAt:    time.Now(),
		Options:      options,
	})
	return err == nil, err
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

// MarkComplete marks a task as complete and stores metadata
func (d *DB) MarkComplete(gid, filePath string, totalLength int64) error {
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

		state.Status = "complete"
		state.FilePath = filePath
		state.TotalLength = totalLength
		state.UpdatedAt = time.Now()

		newData, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return b.Put([]byte(gid), newData)
	})
}

// GetStoppedTasks returns tasks that are complete or in error state, sorted by updated time descending
func (d *DB) GetStoppedTasks(offset, num int) ([]UploadState, int, error) {
	var uploads []UploadState
	err := d.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		return b.ForEach(func(k, v []byte) error {
			var state UploadState
			if err := json.Unmarshal(v, &state); err != nil {
				return nil
			}
			if state.Status == "complete" || state.Status == "error" || state.Status == "removed" {
				uploads = append(uploads, state)
			}
			return nil
		})
	})
	if err != nil {
		return nil, 0, err
	}

	// Sort by UpdatedAt descending
	sort.Slice(uploads, func(i, j int) bool {
		return uploads[i].UpdatedAt.After(uploads[j].UpdatedAt)
	})

	total := len(uploads)
	if offset >= total {
		return []UploadState{}, total, nil
	}
	end := offset + num
	if end > total {
		end = total
	}

	return uploads[offset:end], total, nil
}

// DeleteTask removes a task from the database
func (d *DB) DeleteTask(gid string) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		return b.Delete([]byte(gid))
	})
}

// PurgeTasks removes all stopped tasks (complete, error, removed) from the database
func (d *DB) PurgeTasks() error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketUploads))
		return b.ForEach(func(k, v []byte) error {
			var state UploadState
			if err := json.Unmarshal(v, &state); err != nil {
				return nil
			}
			if state.Status == "complete" || state.Status == "error" || state.Status == "removed" {
				return b.Delete(k)
			}
			return nil
		})
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
