package files

import (
	"fmt"
	"github.com/1Panel-dev/1Panel/backend/buserr"
	"github.com/1Panel-dev/1Panel/backend/constant"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
)

type FileInfo struct {
	Fs         afero.Fs    `json:"-"`
	Path       string      `json:"path"`
	Name       string      `json:"name"`
	User       string      `json:"user"`
	Group      string      `json:"group"`
	Extension  string      `json:"extension"`
	Content    string      `json:"content"`
	Size       int64       `json:"size"`
	IsDir      bool        `json:"isDir"`
	IsSymlink  bool        `json:"isSymlink"`
	IsHidden   bool        `json:"isHidden"`
	LinkPath   string      `json:"linkPath"`
	Type       string      `json:"type"`
	Mode       string      `json:"mode"`
	MimeType   string      `json:"mimeType"`
	UpdateTime time.Time   `json:"updateTime"`
	ModTime    time.Time   `json:"modTime"`
	FileMode   os.FileMode `json:"-"`
	Items      []*FileInfo `json:"items"`
	ItemTotal  int         `json:"itemTotal"`
}

type FileOption struct {
	Path       string `json:"path"`
	Search     string `json:"search"`
	ContainSub bool   `json:"containSub"`
	Expand     bool   `json:"expand"`
	Dir        bool   `json:"dir"`
	ShowHidden bool   `json:"showHidden"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
}

type FileSearchInfo struct {
	Path string `json:"path"`
	fs.FileInfo
}

func NewFileInfo(op FileOption) (*FileInfo, error) {
	var appFs = afero.NewOsFs()

	info, err := appFs.Stat(op.Path)
	if err != nil {
		return nil, err
	}

	file := &FileInfo{
		Fs:        appFs,
		Path:      op.Path,
		Name:      info.Name(),
		IsDir:     info.IsDir(),
		FileMode:  info.Mode(),
		ModTime:   info.ModTime(),
		Size:      info.Size(),
		IsSymlink: IsSymlink(info.Mode()),
		Extension: filepath.Ext(info.Name()),
		IsHidden:  IsHidden(op.Path),
		Mode:      fmt.Sprintf("%04o", info.Mode().Perm()),
		MimeType:  GetMimeType(op.Path),
	}
	if file.IsSymlink {
		file.LinkPath = GetSymlink(op.Path)
	}
	if op.Expand {
		if file.IsDir {
			if err := file.listChildren(op.Dir, op.ShowHidden, op.ContainSub, op.Search, op.Page, op.PageSize); err != nil {
				return nil, err
			}
			return file, nil
		} else {
			if err := file.getContent(); err != nil {
				return nil, err
			}
		}
	}
	return file, nil
}

func (f *FileInfo) search(dir, showHidden bool, af afero.Afero, search string, count int) ([]FileSearchInfo, error) {
	var files []FileSearchInfo
	if err := afero.Walk(af, f.Path, func(path string, info fs.FileInfo, err error) error {
		if info != nil {

			if dir && !info.IsDir() {
				return nil
			}
			if !showHidden && IsHidden(info.Name()) {
				return nil
			}

			lowerName := strings.ToLower(info.Name())
			lowerSearch := strings.ToLower(search)
			if strings.Contains(lowerName, lowerSearch) {
				files = append(files, FileSearchInfo{
					Path:     path,
					FileInfo: info,
				})
				if len(files) > count {
					return nil
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func (f *FileInfo) listChildren(dir, showHidden, containSub bool, search string, page, pageSize int) error {
	afs := &afero.Afero{Fs: f.Fs}
	var (
		files []FileSearchInfo
		err   error
	)

	if search != "" && containSub {
		files, err = f.search(dir, showHidden, *afs, search, page*pageSize)
		if err != nil {
			return err
		}
	} else {
		dirFiles, err := afs.ReadDir(f.Path)
		if err != nil {
			return err
		}
		for _, file := range dirFiles {
			files = append(files, FileSearchInfo{
				Path:     f.Path,
				FileInfo: file,
			})
		}
	}

	f.ItemTotal = 0

	var items []*FileInfo
	for _, df := range files {
		if dir && !df.IsDir() {
			continue
		}

		name := df.Name()
		fPath := path.Join(df.Path, df.Name())

		if search != "" {
			if containSub {
				fPath = df.Path
				name = strings.TrimPrefix(strings.TrimPrefix(fPath, f.Path), "/")
			} else {
				lowerName := strings.ToLower(name)
				lowerSearch := strings.ToLower(search)
				if !strings.Contains(lowerName, lowerSearch) {
					continue
				}
			}
		}

		if !showHidden && IsHidden(name) {
			continue
		}
		f.ItemTotal++

		isSymlink, isInvalidLink := false, false
		if IsSymlink(df.Mode()) {
			isSymlink = true
			info, err := f.Fs.Stat(fPath)
			if err == nil {
				df.FileInfo = info
			} else {
				isInvalidLink = true
			}
		}

		file := &FileInfo{
			Fs:        f.Fs,
			Name:      name,
			Size:      df.Size(),
			ModTime:   df.ModTime(),
			FileMode:  df.Mode(),
			IsDir:     df.IsDir(),
			IsSymlink: isSymlink,
			IsHidden:  IsHidden(fPath),
			Extension: filepath.Ext(name),
			Path:      fPath,
			Mode:      fmt.Sprintf("%04o", df.Mode().Perm()),
			MimeType:  GetMimeType(fPath),
		}

		if isSymlink {
			file.LinkPath = GetSymlink(fPath)
		}

		if isInvalidLink {
			file.Type = "invalid_link"
		}
		items = append(items, file)
	}

	start := (page - 1) * pageSize
	end := pageSize + start
	var result []*FileInfo
	if start < 0 || start > f.ItemTotal || end < 0 || start > end {
		result = items
	} else {
		if end > f.ItemTotal {
			result = items[start:]
		} else {
			result = items[start:end]
		}
	}

	f.Items = result
	return nil
}

func (f *FileInfo) getContent() error {
	if f.Size <= 10*1024*1024 {
		afs := &afero.Afero{Fs: f.Fs}
		cByte, err := afs.ReadFile(f.Path)
		if err != nil {
			return nil
		}
		if len(cByte) > 0 && detectBinary(cByte) {
			return buserr.New(constant.ErrFileCanNotRead)
		}
		f.Content = string(cByte)
		return nil
	} else {
		return buserr.New(constant.ErrFileCanNotRead)
	}
}

func detectBinary(buf []byte) bool {
	whiteByte := 0
	n := min(1024, len(buf))
	for i := 0; i < n; i++ {
		if (buf[i] >= 0x20) || buf[i] == 9 || buf[i] == 10 || buf[i] == 13 {
			whiteByte++
		} else if buf[i] <= 6 || (buf[i] >= 14 && buf[i] <= 31) {
			return true
		}
	}

	return whiteByte < 1
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
