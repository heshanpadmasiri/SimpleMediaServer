package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type File struct {
	name string
	id   int
}

type Directory struct {
	name           string
	files          []File
	childDirectory []Directory
}

type Context struct {
	paths []string
}

// Make this a method of Context
func getPath(cx *Context, file *File) string {
	return cx.paths[file.id]
}

func splitPath(path string) (string, string) {
	split := strings.SplitN(path, "/", 2)
	if len(split) == 1 {
		return split[0], ""
	}
	return split[0], split[1]
}

func getDirectoryByPath(root *Directory, path string) *Directory {
	path = strings.Trim(path, "/")
	if path == "" {
		return root
	}
	dirName, rest := splitPath(path)
	for _, dir := range root.childDirectory {
		if dir.name == dirName {
			return getDirectoryByPath(&dir, rest)
		}
	}
	return nil
}

func addFileToContext(cx *Context, path string) (File, error) {
	cx.paths = append(cx.paths, path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return File{}, err
	}

	name := fileInfo.Name()
	return File{name: name, id: len(cx.paths) - 1}, nil
}

func filteredFile(path string) bool {
	return filepath.Base(path)[0] == '.'
}

func addDirRootToContext(cx *Context, path string) (Directory, error) {
	childDirectory := make([]Directory, 0)
	files := make([]File, 0)
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if filePath == path || filteredFile(filePath) {
			return nil
		}
		if err != nil {
			return err
		}

		if info.IsDir() {
			dir, err := addDirRootToContext(cx, filePath)
			if err != nil {
				return err
			}
			childDirectory = append(childDirectory, dir)
		} else {
			file, err := addFileToContext(cx, filePath)
			if err != nil {
				return err
			}
			files = append(files, file)
		}

		return nil
	})

	if err != nil {
		return Directory{}, err
	}
	name := filepath.Base(path)
	return Directory{childDirectory: childDirectory, files: files, name: name}, nil
}

func main() {
	cx := Context{paths: make([]string, 0)}
	dir, err := addDirRootToContext(&cx, "./testData")
	if err != nil {
		panic(err)
	}
	r := gin.Default()
	r.GET("/ping/*path", func(c *gin.Context) {
		path := c.Param("path")
		directory := getDirectoryByPath(&dir, path)
		if directory == nil {
			c.JSON(404, gin.H{
				"message": "Not found",
			})
			return
		}
		c.JSON(200, gin.H{
			"message": "pong",
			"path":    path,
			"name": directory.name,
		})
	})

	fmt.Println(dir)
	r.Run()
}
