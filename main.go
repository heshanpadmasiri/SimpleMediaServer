package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type FileKind int

const (
	Image FileKind = iota
	Video
	Other
)

type File struct {
	name string
	id   int
	kind FileKind
}

type Directory struct {
	name           string
	files          []File
	childDirectory []Directory
}

type Context struct {
	paths           []string
	fileCacheReady  bool
	mu              sync.Mutex
	thumbnailPaths  map[int]string
	nextThumbnailId int
}

// given file index get the path from context
func (cx *Context) getPath(id int) (string, error) {
	if id < 0 || id >= len(cx.paths) {
		return "", fmt.Errorf("invalid id %d", id)
	}
	return cx.paths[id], nil
}

func (cx *Context) getVideoThumbnailPathFor(id int) (string, error) {
	fmt.Printf("Getting thumbnail for video %d\n", id)
	cx.mu.Lock()
	fmt.Printf("locked for %d\n", id)
	defer cx.mu.Unlock()
	if !cx.fileCacheReady {
		fmt.Printf("initializing cache for %d\n", id)
		err := cx.initFileCache()
		if err != nil {
			return "", err
		}
	}
	if path, ok := cx.thumbnailPaths[id]; ok {
		fmt.Printf("thumbnail already generated for %d\n", id)
		return path, nil
	}
	videoPath, err := cx.getPath(id)
	if err != nil {
		return "", err
	}
	return cx.generateThumbnailForVideo(videoPath)
}

func (cx *Context) getNextThumbnailPath() string {
	cx.nextThumbnailId++
	return fmt.Sprintf("/cache/thumbnail%d.jpg", cx.nextThumbnailId)
}

func (cx *Context) cleanCache() {
	exec.Command("rm", "-rf", "./cache").Output()
}

func (cx *Context) generateThumbnailForVideo(videoPath string) (string, error) {
	// generate a thumbnail for the video
	thumbnailPath := cx.getNextThumbnailPath()
	cmd := exec.Command("ffmpeg", "-i", videoPath, "-ss", "00:00:01.000", "-vframes", "1", thumbnailPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error executing ffmpeg:", err)
		fmt.Println("ffmpeg output:", string(output))
		return "", err
	}
	return thumbnailPath, nil
}

func (cx *Context) initFileCache() error {
	// check if a directory called "cache" exists in the current directory, if so delete it
	// create a new directory called "cache"
	_, err := os.Stat("cache")
	if os.IsNotExist(err) {
		err := os.Mkdir("cache", 0755)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	cx.thumbnailPaths = make(map[int]string)
	cx.fileCacheReady = true
	return nil
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
	return File{name: name, id: len(cx.paths) - 1, kind: fileKind(path)}, nil
}

func fileKind(path string) FileKind {
	ext := filepath.Ext(path)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return Image
	case ".mp4", ".webm":
		return Video
	default:
		return Other
	}
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
	ThumnailUrl string
	IsVideo     bool
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

func slideUrl(path string, file File) string {
	prefix := "/slides/"
	basePath := strings.Trim(path, "/")
	id := strconv.Itoa(file.id)
	if basePath == "" {
		return prefix + id
	}
	return prefix + id + "/" + basePath
}

func fileResourceUrl(file File) string {
	switch file.kind {
	case Video:
		return videoResourceUrlById(file.id)
	case Image:
		return imageResourceUrlById(file.id)
	default:
		panic("unimplemented")
	}
}

func fileThumbnailUrl(cx *Context, file File) string {
	switch file.kind {
	case Video:
		path, error := cx.getVideoThumbnailPathFor(file.id)
		if error != nil {
			return ""
		}
		return path
	case Image:
		return imageResourceUrlById(file.id)
	default:
		panic("unimplemented")
	}
}

func videoResourceUrlById(id int) string {
	return "/video/" + strconv.Itoa(id)
}

func imageResourceUrlById(id int) string {
	return "/img/" + strconv.Itoa(id)
}

func fileData(cx *Context, directory *Directory, path string) []FileData {
	data := make([]FileData, 0)
	for _, file := range directory.files {
		if file.kind == Other {
			continue
		}
		data = append(data, FileData{Name: file.name, Url: slideUrl(path, file), ResourceUrl: fileResourceUrl(file), ThumnailUrl: fileThumbnailUrl(cx, file)})
	}
	return data
}

func main() {
	cx := Context{paths: make([]string, 0)}
	cx.cleanCache()
	dataPath := os.Args[1]
	dir, err := addDirRootToContext(&cx, dataPath)
	if err != nil {
		panic(err)
	}
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/", func(c *gin.Context) {
		returnDirectoryPage(c, &cx, &dir, "")
	})
	r.GET("/files/*path", func(c *gin.Context) {
		path := c.Param("path")
		directory := getDirectoryByPath(&dir, path)
		if directory == nil {
			c.HTML(http.StatusNotFound, "invalidPath.tmpl", gin.H{
				"path": path,
			})
			return
		}
		returnDirectoryPage(c, &cx, directory, path)
	})
	// TODO: refactor image and vidoe handlers
	r.GET("/img/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
				"reason": err,
			})
			return
		}
		returnFileById(&cx, c, id)
	})
	r.GET("/cache/:name", func(c *gin.Context) {
		returnFileByPath(c, fmt.Sprintf("cache/%s", c.Param("name")))
	})
	r.GET("/video/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
				"reason": err,
			})
			return
		}
		returnFileById(&cx, c, id)
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
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
				"reason": err,
			})
			return
		}
		names := ""
		index := index(directory.files, id)
		isVideo := directory.files[index].kind == Video
		files := fileData(&cx, directory, path)
		others := getFilesInRange(files, index)
		prev := prevUrl(directory.files, index, path)
		next := nextUrl(directory.files, index, path)
		resourceUrl := imageResourceUrlById(id)
		c.HTML(http.StatusOK, "slide.tmpl", gin.H{
			"Name":        names,
			"isVideo":     isVideo,
			"ResourceUrl": resourceUrl,
			"PrevUrl":     prev,
			"NextUrl":     next,
			"Others":      others,
		})
	})

	fmt.Println(dir)
	r.Run()
}

func getFilesInRange(files []FileData, index int) []FileData {
	numFiles := len(files)
	if numFiles <= 11 {
		return files
	}

	// Calculate the range of files to include
	start := index - 5
	end := index + 5

	// Handle wraparound if necessary
	if start < 0 {
		start += numFiles
	}
	if end >= numFiles {
		end -= numFiles
	}

	// Create a new slice with the files in the range
	var result []FileData
	if start <= end {
		result = files[start : end+1]
	} else {
		result = append(files[start:], files[:end+1]...)
	}

	return result
}

func index(files []File, id int) int {
	for i, file := range files {
		if file.id == id {
			return i
		}
	}
	return -1
}

func nextUrl(files []File, i int, path string) string {
	if i+1 < len(files) {
		return slideUrl(path, files[i+1])
	} else {
		log.Println(files[0])
		return slideUrl(path, files[0])
	}
}

func prevUrl(files []File, i int, path string) string {
	if i-1 >= 0 {
		return slideUrl(path, files[i-1])
	} else {
		return slideUrl(path, files[len(files)-1])
	}
}

func returnDirectoryPage(c *gin.Context, cx *Context, directory *Directory, path string) {
	Directories := childDirectoryData(directory, path)
	files := fileData(cx, directory, path)
	if len(files) > 10 {
		files = files[:10]
	}
	c.HTML(http.StatusOK, "directoryData.tmpl", gin.H{
		"name":        directory.name,
		"Directories": Directories,
		"Files":       files,
	})
}

func returnFileByPath(c *gin.Context, path string) {
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

func returnFileById(cx *Context, c *gin.Context, id int) {
	path, err := cx.getPath(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
			"reason": err,
		})
		return
	}
	returnFileByPath(c, path)
}
