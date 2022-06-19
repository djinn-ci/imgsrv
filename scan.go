package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type driverGroup struct {
	name string
	re   *regexp.Regexp
}

type driver struct {
	name       string
	categories map[string]struct{}
	groups     []driverGroup
}

type Scanner struct {
	dir     string
	errh    func(error)
	drivers map[string]driver
}

func (s *Scanner) driverHasCategory(driver, category string) bool {
	if driver, ok := s.drivers[driver]; ok {
		if _, ok := driver.categories[category]; ok {
			return true
		}
	}
	return false
}

func (s *Scanner) Scan() []*Image {
	symlinks := make(map[string]struct{})

	initial := make([]*Image, 0)

	err := filepath.Walk(s.dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		modtime := info.ModTime()

		relpath := strings.Replace(path, s.dir+string(os.PathSeparator), "", 1)
		parts := strings.Split(relpath, string(os.PathSeparator))

		var (
			driver   driver
			ok       bool
			category string
		)

		if len(parts) > 1 {
			driver, ok = s.drivers[parts[0]]

			if !ok {
				s.errh(errors.New("scan: " + path + " - invalid driver " + parts[0]))
				return nil
			}

			parts = parts[1:]

			if len(parts) >= 1 {
				if s.driverHasCategory(driver.name, parts[0]) {
					category = parts[0]
					parts = parts[1:]
				}
			}

			name := strings.Join(parts, string(os.PathSeparator))

			var link, group string

			if info.Mode().Type() == fs.ModeSymlink {
				link, _ = os.Readlink(path)

				linkpath := filepath.Join(filepath.Dir(path), link)

				symlinks[linkpath] = struct{}{}

				info, err := os.Stat(linkpath)

				if err != nil {
					return err
				}
				modtime = info.ModTime()
			}

			for _, grp := range driver.groups {
				if grp.re.Match([]byte(name)) {
					group = grp.name
					break
				}
			}

			if link != "" {
				link = filepath.Join(filepath.Dir(name), link)
			}

			initial = append(initial, &Image{
				Path:     path,
				Driver:   driver.name,
				Category: category,
				Group:    group,
				Name:     name,
				Link:     link,
				ModTime:  modtime,
			})
		}
		return nil
	})

	if err != nil {
		s.errh(err)
	}

	imgs := make([]*Image, 0, len(initial))

	for _, img := range initial {
		if _, ok := symlinks[img.Path]; ok {
			continue
		}
		imgs = append(imgs, img)
	}
	return imgs
}
