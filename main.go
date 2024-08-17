package main

import (
	"fmt"
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

type DirectoryData struct {
	Name string
	Url  string
}

type FileData struct {
	Name        string
	Url         string
	ResourceUrl string
}

func directoryUrl(path, name string) string {
	basePath := strings.Trim(path, "/")
	prefix := "/files/"
	if basePath == "" {
		return prefix + name
	}
	return prefix + basePath + "/" + name
}

func childDirectoryData(directory *Directory, path string) []DirectoryData {
	data := make([]DirectoryData, 0)
	for _, dir := range directory.childDirectory {
		data = append(data, DirectoryData{Name: dir.name, Url: directoryUrl(path, dir.name)})
	}
	return data
}

func fileUrl(path string, file File) string {
	prefix := "/slides/"
	basePath := strings.Trim(path, "/")
	id := strconv.Itoa(file.id)
	if basePath == "" {
		return prefix + id
	}
	return prefix + id + "/" + basePath
}

func fileResourceUrl(file File) string {
	return fileResourceUrlById(file.id)
}

func fileResourceUrlById(id int) string {
	return "/img/" + strconv.Itoa(id)
}

func fileData(directory *Directory, path string) []FileData {
	data := make([]FileData, 0)
	for _, file := range directory.files {
		data = append(data, FileData{Name: file.name, Url: fileUrl(path, file), ResourceUrl: fileResourceUrl(file)})
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
		directory := getDirectoryByPath(&dir, path)
		if directory == nil {
			c.HTML(http.StatusNotFound, "invalidPath.tmpl", gin.H{
				"path": path,
			})
			return
		}
		Directories := childDirectoryData(directory, path)
		Files := fileData(directory, path)
		c.HTML(http.StatusOK, "directoryData.tmpl", gin.H{
			"name":        directory.name,
			"Directories": Directories,
			"Files":       Files,
		})
	})
	r.GET("/img/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
				"reason": err,
			})
			return
		}
		returnImageById(&cx, c, id)
	})
	r.GET("/slides/:id/*path", func(c *gin.Context) {
		path := c.Param("path")
		idStr := c.Param("id")
		directory := getDirectoryByPath(&dir, path)
		if directory == nil {
			c.HTML(http.StatusNotFound, "invalidPath.tmpl", gin.H{
				"path": path,
			})
			return
		}
		files := fileData(directory, path)
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
				"reason": err,
			})
			return
		}
		names := ""
		prev := prevUrl(dir.files, id)
		next := nextUrl(dir.files, id)
		resourceUrl := fileResourceUrlById(id)
		c.HTML(http.StatusOK, "slide.tmpl", gin.H{
			"Name":        names,
			"ResourceUrl": resourceUrl,
			"PrevUrl":     prev,
			"NextUrl":     next,
			"Others":      files,
		})
	})

	fmt.Println(dir)
	r.Run()
}

func nextUrl(files []File, id int) string {
	for i, file := range files {
		if file.id == id {
			if i+1 < len(files) {
				return fileResourceUrl(files[i+1])
			} else {
				return fileResourceUrl(files[0])
			}
		}
	}
	return ""
}

func prevUrl(files []File, id int) string {
	for i, file := range files {
		if file.id == id {
			if i-1 >= 0 {
				return fileResourceUrl(files[i-1])
			} else {
				return fileResourceUrl(files[len(files)-1])
			}
		}
	}
	return ""
}

func returnImageById(cx *Context, c *gin.Context, id int) {
	path, err := getPath(cx, id)
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
}
