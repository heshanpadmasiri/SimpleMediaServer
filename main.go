package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
func getPath(cx *Context, id int) (string, error) {
	if id < 0 || id >= len(cx.paths) {
		return "", fmt.Errorf("invalid id %d", id)
	}
	return cx.paths[id], nil
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

type FileData struct {
	Name string
	Url string
}

func direcotryUrl(path, name string) string {
	basePath := strings.Trim(path, "/")
	if basePath == "" {
		return "/files/" + name
	}
	return "/files/" + basePath + "/" + name
}
func childDirectoryData(directory *Directory, path string) []FileData {
	data := make([]FileData, 0)
	for _, dir := range directory.childDirectory {
		data = append(data, FileData{Name: dir.name, Url: direcotryUrl(path, dir.name)})
	}
	return data
}

func fileUrl(directory *Directory, file File) string {
	return "/img/" + directory.name + "/" + strconv.Itoa(file.id)
}

func fileData(directory *Directory) []FileData {
	data := make([]FileData, 0)
	for _, file := range directory.files {
		data = append(data, FileData{Name: file.name, Url: fileUrl(directory, file)})
	}
	return data
}

func main() {
	cx := Context{paths: make([]string, 0)}
	dir, err := addDirRootToContext(&cx, "./testData")
	if err != nil {
		panic(err)
	}
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/files/*path", func(c *gin.Context) {
		path := c.Param("path")
		log.Println(path)
		directory := getDirectoryByPath(&dir, path)
		log.Println(directory)
		if directory == nil {
			c.HTML(http.StatusNotFound, "invalidPath.tmpl", gin.H{
				"path": path,
			})
			return
		}
		Directories := childDirectoryData(directory, path)
		Files := fileData(directory)
		c.HTML(http.StatusOK, "directoryData.tmpl", gin.H{
			"name": directory.name,
			"Directories": Directories,
			"Files": Files,
		})
	})
	r.GET("/img/:directory/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
				"reason": err,
			})
			return
		}
		path, err := getPath(&cx, id)
		if err != nil {
			c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
				"reason": err,
			})
			return
		}
		file, err := os.Open(path)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"reason": err,
			})
			return
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"reason": err,
			})
			return
		}

		fileSize := fileInfo.Size()
		buffer := make([]byte, fileSize)

		_, err = file.Read(buffer)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"reason": err,
			})
			return
		}

		contentType := http.DetectContentType(buffer)
		c.Data(http.StatusOK, contentType, buffer)
	})

	fmt.Println(dir)
	r.Run()
}
