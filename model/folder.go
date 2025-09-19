package model

import "fmt"

type FileSystem struct {
	Name     string
	FullPath string
	Size     int64
	IsDir    bool
}

func (f *FileSystem) GetFormattedSize() string {
	switch {
	case f.Size < 1024:
		return fmt.Sprintf("%d bytes", f.Size)
	case f.Size < 1024*1024:
		return fmt.Sprintf("%.2f KB", float64(f.Size)/1024)
	case f.Size < 1024*1024*1024:
		return fmt.Sprintf("%.2f MB", float64(f.Size)/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", float64(f.Size)/(1024*1024*1024))
	}
}

type Directory struct {
	FileSystem
	SubDirs []Directory
	Files   []FileSystem
}
