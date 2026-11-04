package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// A restore is staged privately and published by one directory rename. Readers
// see either the entire imported batch or none of it. Existing files stay put.
func libraryLocations(root, kind string) ([]string, error) {
	locations := []string{}
	add := func(path string) error {
		st, err := os.Lstat(path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("library directory must not be a link: %s", kind)
		}
		locations = append(locations, path)
		return nil
	}
	if err := add(filepath.Join(root, kind)); err != nil {
		return nil, err
	}
	batches := filepath.Join(root, "restores")
	st, err := os.Lstat(batches)
	if os.IsNotExist(err) {
		return locations, nil
	}
	if err != nil {
		return nil, err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("restore directory must be an owned directory")
	}
	dirs, err := os.ReadDir(batches)
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		if !managedID.MatchString(d.Name()) {
			continue
		}
		if !d.IsDir() {
			return nil, fmt.Errorf("invalid restored library directory")
		}
		if err := add(filepath.Join(batches, d.Name(), kind)); err != nil {
			return nil, err
		}
	}
	return locations, nil
}

func libraryEntryDirectory(root, kind, id string) string {
	locations, err := libraryLocations(root, kind)
	if err == nil {
		for _, location := range locations {
			p := filepath.Join(location, id)
			st, e := os.Lstat(p)
			if e == nil && st.IsDir() && st.Mode()&os.ModeSymlink == 0 {
				return p
			}
		}
	}
	return filepath.Join(root, kind, id)
}
