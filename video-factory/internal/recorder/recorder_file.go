package recorder

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
	"time"
)

type Pattern struct {
	Username   string
	Year       string
	Month      string
	Day        string
	Hour       string
	Minute     string
	Second     string
	Sequence   int
	RoomRealId string
	Ext        string
}

var (
	sequencePatternRegex = regexp.MustCompile(`\{\{\s*\.Sequence\s*}}`)
	extPatternRegex      = regexp.MustCompile(`\{\{\s*\.Ext\s*}}`)
)

func (r *Recorder) GenerateFileName() (string, error) {
	var buf bytes.Buffer

	filenamePattern := r.Config.Recorder.FilenamePattern

	// 必须包含 sequence，否则文件会覆盖
	if !sequencePatternRegex.MatchString(filenamePattern) {
		filenamePattern += "_{{.Sequence}}"
	}
	if !extPatternRegex.MatchString(filenamePattern) {
		filenamePattern += ".{{.Ext}}"
	}

	tpl, err := template.New("filename").Parse(filenamePattern)
	if err != nil {
		return "", fmt.Errorf("filename pattern error: %w", err)
	}

	t := time.Unix(r.StreamAt, 0)
	pattern := &Pattern{
		Username:   r.Username,
		Year:       t.Format("2006"),
		Month:      t.Format("01"),
		Day:        t.Format("02"),
		Hour:       t.Format("15"),
		Minute:     t.Format("04"),
		Second:     t.Format("05"),
		Sequence:   r.Sequence,
		RoomRealId: r.RoomRealId,
		Ext:        r.Ext,
	}

	if err = tpl.Execute(&buf, pattern); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return buf.String(), nil
}

func (r *Recorder) InitialSequence() error {
	for i := 0; i < 1000; i++ {
		r.Sequence = i

		filename, err := r.GenerateFileName()
		if err != nil {
			return err
		}

		_, err = os.Stat(filename)
		if os.IsNotExist(err) {
			// file not exist, so sequence is available
			return nil
		}
	}

	return fmt.Errorf("failed to find available sequence number")
}

// ShouldSwitchFile determine if the file should be switched
func (r *Recorder) ShouldSwitchFile() bool {
	maxFilesizeBytes := r.Config.Recorder.MaxFilesize * 1024 * 1024
	maxDurationSeconds := r.Config.Recorder.MaxDuration * 60

	return (r.Duration >= float64(maxDurationSeconds) && r.Config.Recorder.MaxDuration > 0) ||
		(r.Filesize >= maxFilesizeBytes && r.Config.Recorder.MaxFilesize > 0)
}

func (r *Recorder) CreateNewFile(filename string) error {
	// Ensure the directory exists before creating the File
	if err := os.MkdirAll(filepath.Dir(filename), 0777); err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}

	// Open the File in append mode, create it if it doesn't exist
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("cannot open File: %s: %w", filename, err)
	}

	r.File = file
	return nil
}

// NextFile prepares the next file to be created, by cleaning up the last file and generating a new one
func (r *Recorder) NextFile() error {
	if err := r.Cleanup(); err != nil {
		return err
	}

	// check the sequence if exist
	if err := r.InitialSequence(); err != nil {
		return fmt.Errorf("initial sequence: %w", err)
	}

	filename, err := r.GenerateFileName()
	if err != nil {
		return err
	}
	if err := r.CreateNewFile(filename); err != nil {
		return err
	}

	// Increment the sequence number for the next file
	r.Sequence++
	return nil
}

// Cleanup cleans the file and resets it, called when the stream errors out or before next file was created.
func (r *Recorder) Cleanup() error {
	if r.File == nil {
		return nil
	}
	filename := r.File.Name()

	defer func() {
		r.Filesize = 0
		r.Duration = 0
	}()

	// Sync the file to ensure data is written to disk
	if err := r.File.Sync(); err != nil && !errors.Is(err, os.ErrClosed) {
		return fmt.Errorf("sync file: %w", err)
	}
	if err := r.File.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return fmt.Errorf("close file: %w", err)
	}

	// Delete the empty file
	fileInfo, err := os.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat file delete zero file: %w", err)
	}
	if fileInfo != nil && fileInfo.Size() == 0 {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("remove zero file: %w", err)
		}
	}
	return nil
}
