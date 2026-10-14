package workbench

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

type preparedFile struct {
	path, temp      string
	data            []byte
	replace         bool
	destination     os.FileInfo
	destinationHash [32]byte
}
type projectPublishFunc func(*preparedFile) error

func prepareFileData(ctx context.Context, data []byte, path, sourcePath string, disk [32]byte, overwrite bool, prefix string, progress func(string, int)) (_ *preparedFile, err error) {
	prepared := &preparedFile{path: path, data: data}
	progress("checking-destination", len(data))
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return nil, fileError(fmt.Errorf("file destination must be a regular file, not a link or directory"))
		}
		if path != sourcePath && !overwrite {
			return nil, &actionError{"file-exists", "Destination exists. Choose another path or explicitly allow overwrite."}
		}
		current, err := readDocumentFile(path)
		if err != nil {
			return nil, err
		}
		prepared.destinationHash = sha256.Sum256(current)
		if path == sourcePath && !overwrite && prepared.destinationHash != disk {
			return nil, &actionError{"file-conflict", "The file changed outside ELab. Save As, or explicitly overwrite it."}
		}
		prepared.replace = true
		prepared.destination = info
	} else if !os.IsNotExist(err) {
		return nil, fileError(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	progress("writing", len(data))
	f, err := os.CreateTemp(filepath.Dir(path), prefix)
	if err != nil {
		return nil, fileError(err)
	}
	prepared.temp = f.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(prepared.temp)
		}
	}()
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return nil, fileError(err)
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return nil, fileError(err)
	}
	if err = f.Close(); err != nil {
		return nil, fileError(err)
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return prepared, nil
}

// Once the service marks publishing, cancellation cannot claim to have stopped
// a filesystem operation that may already be replacing the destination.
func publishPreparedFile(prepared *preparedFile) error {
	var err error
	if prepared.replace {
		current, readErr := readDocumentFile(prepared.path)
		if readErr != nil {
			return &actionError{"file-conflict", "File destination changed during preparation; choose the destination again."}
		}
		info, statErr := os.Lstat(prepared.path)
		if statErr != nil || !info.Mode().IsRegular() || !os.SameFile(info, prepared.destination) || sha256.Sum256(current) != prepared.destinationHash {
			return &actionError{"file-conflict", "File destination changed during preparation; choose the destination again."}
		}
		err = os.Rename(prepared.temp, prepared.path)
	} else {
		err = os.Link(prepared.temp, prepared.path)
	}
	if err != nil {
		if os.IsExist(err) {
			return &actionError{"file-exists", "Destination was created during preparation; choose another path."}
		}
		return fileError(err)
	}
	if directory, err := os.Open(filepath.Dir(prepared.path)); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}
