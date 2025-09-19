package service

import (
	"fmt"
	"go-cleaner/model"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const OneGb = 1024 * 1024 * 1024
const ONE_MB = 1024 * 1024
const OneKb = 1024

type FileService interface {
	ListDirectories(path string) (model.Directory, error)
	ListFiles(path string) ([]model.FileSystem, error)
	PrintDirectories(directories []model.Directory)
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
	root, err := getDirectoryRecursively(path)
	if err != nil {
		fmt.Println("Error on dir: "+filepath.Base(path), err)
		return model.Directory{}, err
	}
	return root, nil
}

func getDirectoryRecursively(path string) (model.Directory, error) {
	realFileSys, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Error on dir: "+filepath.Base(path), err)
	}

	dir := model.Directory{}
	var folderSize int64 = 0

	var subDirPaths []string
	for _, file := range realFileSys {
		if !file.IsDir() {
			subFile := toFileSystemModel(path, file)
			dir.Files = append(dir.Files, subFile)
			folderSize += subFile.Size
		} else {
			subDirPaths = append(subDirPaths, filepath.Join(path, file.Name()))
		}
	}

	const maxWorkers = 10
	subDirs := make([]model.Directory, len(subDirPaths))

	if len(subDirPaths) > 0 {
		semaphore := make(chan struct{}, maxWorkers)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for i, subPath := range subDirPaths {
			wg.Add(1)
			go func(index int, path string) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire semaphore
				defer func() { <-semaphore }() // Release semaphore

				subDir, err := getDirectoryRecursively(path)
				if err != nil {
					fmt.Println("Error on dir: "+filepath.Base(path), err)
					return
				}

				mu.Lock()
				subDirs[index] = subDir
				mu.Unlock()
			}(i, subPath)
		}
		wg.Wait()

		// Filter out empty directories (from errors)
		for _, subDir := range subDirs {
			if subDir.Name != "" {
				dir.SubDirs = append(dir.SubDirs, subDir)
			}
		}
	}
	dir = toDirModel(path, dir, folderSize)

	return dir, nil
}

func toDirModel(path string, dir model.Directory, folderSize int64) model.Directory {
	fullPath, _ := filepath.Abs(filepath.Join(path, filepath.Base(path)))
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

func (fs *FileSystemService) PrintDirectories(directories []model.Directory) {
	for _, dir := range directories {
		if dir.Size < fs.limit {
			continue
		}
		fmt.Println("Name:", dir.Name)
		fmt.Println("Path:", dir.FullPath)
		fmt.Println("Size:", dir.GetFormattedSize())
		fmt.Println("-----")

	}

}

func (*FileSystemService) ListFiles(path string) ([]model.FileSystem, error) {
	return []model.FileSystem{}, nil
}

func planeDirectory(m model.Directory, list []model.Directory) []model.Directory {

	list = append(list, m)
	for _, dir := range m.SubDirs {
		list = planeDirectory(dir, list)
	}
	return list
}
func ReorderDirectory(m model.Directory) []model.Directory {

	result := planeDirectory(m, []model.Directory{})
	sort.Slice(result, func(i, j int) bool {
		return result[i].Size > result[j].Size
	})
	return result
}
