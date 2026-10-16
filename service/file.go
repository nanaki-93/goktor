package service

import (
	"fmt"

	"github.com/nanaki-93/goktor/model"

	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	OneGb      = 1024 * 1024 * 1024
	OneMb      = 1024 * 1024
	OneKb      = 1024
	maxWorkers = 10
)

type FileService interface {
	ListDirectories(path string) (model.Directory, error)
	ListDirectoriesWithFilter(path string, filter func(model.Directory) bool) (model.Directory, error)
	ListFiles(path string) ([]model.FileSystem, error)
	PrintDirectories(directories []model.Directory, filter func(model.Directory) bool)
	PrintFiles(files []model.FileSystem)
	GetSizeFilter() func(model.Directory) bool
}
type FileSystemService struct {
	limit     int64
	logger    Logger
	semaphore chan struct{}
	semOnce   sync.Once
}

func NewFileService() FileService {
	return &FileSystemService{
		limit:     OneGb * 10, // 1 GB
		logger:    &DefaultLogger{},
		semaphore: make(chan struct{}, maxWorkers),
	}
}

func NewServiceWithLogger(logger Logger) FileService {
	return &FileSystemService{
		limit:     OneGb * 10,
		logger:    logger,
		semaphore: make(chan struct{}, maxWorkers),
	}
}

func NewServiceWithLimit(limit int64) FileService {
	return &FileSystemService{
		limit:     limit,
		logger:    &DefaultLogger{},
		semaphore: make(chan struct{}, maxWorkers),
	}
}

func (fs *FileSystemService) getSemaphore() chan struct{} {
	fs.semOnce.Do(func() {
		if fs.semaphore == nil {
			fs.semaphore = make(chan struct{}, maxWorkers)
		}
	})
	return fs.semaphore
}

func (fs *FileSystemService) resolveAbsPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	fullPath, err := filepath.Abs(path)
	if err != nil {
		if fs.logger != nil {
			fs.logger.Debug("failed to get absolute path", "path", path, "error", err)
		}
		return path
	}
	return fullPath
}

func (fs *FileSystemService) PrintFiles(files []model.FileSystem) {
	for _, file := range files {
		fmt.Println("Name:", file.Name)
		fmt.Println("Path:", file.FullPath)
		fmt.Println("Size:", file.GetFormattedSize())
		fmt.Println("-----")
	}
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
func (fs *FileSystemService) ListDirectories(path string) (model.Directory, error) {
	return fs.ListDirectoriesWithFilter(path, func(model.Directory) bool { return true })
}

func (fs *FileSystemService) ListDirectoriesWithFilter(path string, filter func(model.Directory) bool) (model.Directory, error) {
	absPath := fs.resolveAbsPath(path)
	root, err := fs.getDirectoryRecursively(absPath, filter)
	if err != nil {
		fs.handleError(err, path)
		return model.Directory{}, err
	}
	return root, nil
}

func (fs *FileSystemService) getDirectoryRecursively(path string, filter func(model.Directory) bool) (model.Directory, error) {
	absPath := fs.resolveAbsPath(path)
	dir, subDirPaths, err := fs.readAndManageDirectory(absPath)
	if err != nil {
		return model.Directory{}, err
	}

	if len(subDirPaths) > 0 {
		dir.SubDirs = fs.processSubDirectories(subDirPaths, filter)
	}

	if filter(dir) {
		return dir, nil
	}
	return model.Directory{}, nil
}

func (fs *FileSystemService) readAndManageDirectory(path string) (model.Directory, []string, error) {
	sem := fs.getSemaphore()
	sem <- struct{}{}
	defer func() { <-sem }()

	entries, err := fs.readDirectory(path)
	if err != nil {
		return model.Directory{}, nil, err
	}

	dir, subDirPaths := fs.manageDirEntries(path, entries)
	return dir, subDirPaths, nil
}

func (fs *FileSystemService) readDirectory(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading directory: %s: %w", path, err)
		}
		return nil, fmt.Errorf("failed to read directory %s: %w", filepath.Base(path), err)
	}
	return entries, nil
}

func (fs *FileSystemService) manageDirEntries(path string, entries []os.DirEntry) (model.Directory, []string) {
	var (
		dir         model.Directory
		subDirPaths []string
		folderSize  int64
	)
	for _, entry := range entries {
		if !entry.IsDir() {
			fileModel := fs.toFileSystemModel(path, entry)
			dir.Files = append(dir.Files, fileModel)
			folderSize += fileModel.Size
		} else {
			subDirPaths = append(subDirPaths, filepath.Join(path, entry.Name()))
		}
	}
	return fs.toDirModel(path, dir, folderSize), subDirPaths
}

func (fs *FileSystemService) processSubDirectories(paths []string, filter func(model.Directory) bool) []model.Directory {
	results := make([]model.Directory, len(paths))

	var wg sync.WaitGroup

	for i, subPath := range paths {
		wg.Add(1)
		go func(index int, subPath string) {
			defer wg.Done()

			subDir, err := fs.getDirectoryRecursively(subPath, filter)
			if err != nil {
				fs.logger.Debug("error processing subdirectory", "path", subPath, "error", err)
				return
			}

			results[index] = subDir
		}(i, subPath)
	}
	wg.Wait()

	return fs.filterResults(results)
}
func (fs *FileSystemService) filterResults(results []model.Directory) []model.Directory {
	var filtered []model.Directory
	for _, dir := range results {
		if dir.Name != "" && dir.FullPath != "" {
			filtered = append(filtered, dir)
		}
	}
	return filtered
}

func (fs *FileSystemService) toDirModel(path string, dir model.Directory, folderSize int64) model.Directory {
	fullPath := fs.resolveAbsPath(path)
	dir.FileSystem.Size = folderSize
	dir.FullPath = fullPath
	dir.IsDir = true
	dir.Name = filepath.Base(fullPath)
	return dir
}

func (fs *FileSystemService) toFileSystemModel(path string, file os.DirEntry) model.FileSystem {
	info, err := file.Info()
	if err != nil {
		fs.logger.Debug("failed to get file info", "file", file, "error", err)
		return model.FileSystem{Name: file.Name()}
	}
	fullPath := filepath.Join(fs.resolveAbsPath(path), file.Name())
	subFile := model.FileSystem{
		Name:     file.Name(),
		FullPath: fullPath,
		Size:     info.Size(),
		IsDir:    file.IsDir(),
	}
	return subFile
}
func (fs *FileSystemService) handleError(err error, path string) {
	if os.IsPermission(err) {
		fs.logger.Error("permission denied reading directory", "path", path)
	} else {
		fs.logger.Error("failed to read directory", "path", filepath.Base(path), "error", err)
	}
}

func (fs *FileSystemService) GetSizeFilter() func(model.Directory) bool {
	return func(dir model.Directory) bool {
		return dir.Size > fs.limit
	}
}

func (fs *FileSystemService) ListFiles(path string) ([]model.FileSystem, error) {
	absPath := fs.resolveAbsPath(path)
	entries, err := fs.readDirectory(absPath)
	if err != nil {
		return nil, err
	}

	var files []model.FileSystem
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, fs.toFileSystemModel(absPath, entry))
		}
	}
	return files, nil
}

func ReorderDirectory(directory model.Directory) []model.Directory {

	result := directory.FlattenDirectory()
	sort.Sort(model.BySize(result))
	return result
}
