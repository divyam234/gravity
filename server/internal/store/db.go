package store

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

const (
	BucketUploads = "uploads"
	DbName        = "aria2-rclone.db"
)

type UploadState struct {
	Gid          string                 `json:"gid"`
	TargetRemote string                 `json:"targetRemote"` // e.g. "gdrive:"
	RcloneJobID  int64                  `json:"rcloneJobId,omitempty"`
	Status       string                 `json:"status"` // pending, uploading, complete, error
	StartedAt    time.Time              `json:"startedAt"`
	LastError    string                 `json:"lastError,omitempty"`
	URIs         []string               `json:"uris,omitempty"`
	Torrent      string                 `json:"torrent,omitempty"`  // Base64 encoded
	Metalink     string                 `json:"metalink,omitempty"` // Base64 encoded
	Options      map[string]interface{} `json:"options,omitempty"`
	RetryCount   int                    `json:"retryCount,omitempty"`
	JobID        string                 `json:"jobId,omitempty"` // Rclone Job ID
	UpdatedAt    time.Time              `json:"updatedAt"`
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
		_, err := tx.CreateBucketIfNotExists([]byte(BucketUploads))
		return err
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

func (d *DB) UpdateStatus(gid string, status string, jobId int64) error {
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
		if jobId != 0 {
			state.RcloneJobID = jobId
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
