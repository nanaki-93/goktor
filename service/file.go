package service

import (
	"fmt"

	"github.com/nanaki-93/goktor/model"

	"os"
	"path/filepath"
	"sort"
	"sync"
)

const OneGb = 1024 * 1024 * 1024
const OneMb = 1024 * 1024
const OneKb = 1024

type FileService interface {
	ListDirectories(path string) (model.Directory, error)
	ListDirectoriesWithFilter(path string, filter func(model.Directory) bool) (model.Directory, error)
	ListFiles(path string) ([]model.FileSystem, error)
	PrintDirectories(directories []model.Directory, filter func(model.Directory) bool)
	GetSizeFilter() func(model.Directory) bool
}
type FileSystemService struct {
	limit int64
}

func NewService() FileService {
	return &FileSystemService{
		limit: OneGb * 10, // 1 GB
	}
}

func (*FileSystemService) ListDirectories(path string) (model.Directory, error) {
	root, err := getDirectoryRecursively(path, func(model.Directory) bool { return true })
	if err != nil {
		if os.IsPermission(err) {
			return model.Directory{}, err
		}
		fmt.Println("Error on dir: "+filepath.Base(path), err)
		return model.Directory{}, err
	}
	return root, nil
}
func (*FileSystemService) ListDirectoriesWithFilter(path string, filter func(model.Directory) bool) (model.Directory, error) {
	root, err := getDirectoryRecursively(path, filter)
	if err != nil {
		if os.IsPermission(err) {
			return model.Directory{}, err
		}
		fmt.Println("Error on dir: "+filepath.Base(path), err)
		return model.Directory{}, err
	}
	return root, nil
}

func getDirectoryRecursively(path string, filter func(model.Directory) bool) (model.Directory, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if !os.IsPermission(err) {
			fmt.Println("Error on dir: "+filepath.Base(path), err)
		}
		return model.Directory{}, err
	}

	dir, subDirPaths := manageDirEntries(path, entries)

	if len(subDirPaths) > 0 {
		dir.SubDirs = processSubDirectories(subDirPaths, filter)
	}

	if filter(dir) {
		return dir, nil
	}
	return model.Directory{}, nil
}

func manageDirEntries(path string, entries []os.DirEntry) (model.Directory, []string) {
	var (
		dir         model.Directory
		subDirPaths []string
		folderSize  int64
	)
	for _, entry := range entries {
		if !entry.IsDir() {
			fileModel := toFileSystemModel(path, entry)
			dir.Files = append(dir.Files, fileModel)
			folderSize += fileModel.Size
		} else {
			subDirPaths = append(subDirPaths, filepath.Join(path, entry.Name()))
		}
	}
	return toDirModel(path, dir, folderSize), subDirPaths
}

func processSubDirectories(paths []string, filter func(model.Directory) bool) []model.Directory {
	const maxWorkers = 10
	results := make([]model.Directory, len(paths))
	semaphore := make(chan struct{}, maxWorkers)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, subPath := range paths {
		wg.Add(1)
		go func(index int, subPath string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			subDir, err := getDirectoryRecursively(subPath, filter)
			if err != nil {
				return
			}

			mu.Lock()
			results[index] = subDir
			mu.Unlock()
		}(i, subPath)
	}
	wg.Wait()

	var filtered []model.Directory
	for _, dir := range results {
		if dir.Name != "" && filter(dir) {
			filtered = append(filtered, dir)
		}
	}
	return filtered
}

func toDirModel(path string, dir model.Directory, folderSize int64) model.Directory {
	fullPath, _ := filepath.Abs(path)
	dir.FileSystem.Size = folderSize
	dir.FullPath = fullPath
	dir.IsDir = true
	dir.Name = filepath.Base(path)
	return dir
}

func toFileSystemModel(path string, file os.DirEntry) model.FileSystem {
	info, _ := file.Info()
	fullPath, _ := filepath.Abs(filepath.Join(path, file.Name()))

	subFile := model.FileSystem{
		Name:     file.Name(),
		FullPath: fullPath,
		Size:     info.Size(),
		IsDir:    file.IsDir(),
	}
	return subFile
}

func (fs *FileSystemService) PrintDirectories(directories []model.Directory, filter func(model.Directory) bool) {
	for _, dir := range directories {
		if filter(dir) {
			fmt.Println("Name:", dir.Name)
			fmt.Println("Path:", dir.FullPath)
			fmt.Println("Size:", dir.GetFormattedSize())
			fmt.Println("-----")
		}
	}
}
func (fs *FileSystemService) GetSizeFilter() func(model.Directory) bool {
	return func(dir model.Directory) bool {
		return dir.Size > fs.limit
	}
}

func WithGitFile(file model.Directory) bool {
	for _, d := range file.SubDirs {
		if d.Name == ".git" {
			return true
		}
	}
	return false
}

func (*FileSystemService) ListFiles(path string) ([]model.FileSystem, error) {
	//todo implementation
	return []model.FileSystem{}, nil
}

func ReorderDirectory(directory model.Directory) []model.Directory {

	result := directory.FlattenDirectory()
	sort.Sort(model.BySize(result))
	return result
}
