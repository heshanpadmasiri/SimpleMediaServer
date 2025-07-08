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
	id             int
	name           string
	files          []File
	childDirectory []Directory
}

type Context struct {
	paths           []string
	directories     []Directory
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
	cx.mu.Lock()
	defer cx.mu.Unlock()
	if !cx.fileCacheReady {
		err := cx.initFileCache()
		if err != nil {
			return "", err
		}
	}
	if path, ok := cx.thumbnailPaths[id]; ok {
		return path, nil
	}
	return cx.generateThumbnailForVideo(id)
}

func (cx *Context) getNextThumbnailPath() string {
	cx.nextThumbnailId++
	return fmt.Sprintf("/cache/thumbnail%d.jpg", cx.nextThumbnailId)
}

func (cx *Context) getNextDirectoryId() int {
	return len(cx.directories)
}

func (cx *Context) getDirectoryById(id int) (*Directory, error) {
	if id < 0 || id >= len(cx.directories) {
		return nil, fmt.Errorf("invalid directory id %d", id)
	}
	return &cx.directories[id], nil
}

func (cx *Context) cleanCache() {
	exec.Command("rm", "-rf", "./cache").Output()
}

func (cx *Context) generateThumbnailForVideo(id int) (string, error) {
	videoPath, err := cx.getPath(id)
	if err != nil {
		return "", err
	}
	// TODO: allow for concurrent generation of thumbnails
	// -- Currently we can't do this becuase we have a lock at the begining of image generation
	path, err := cx.generateThumbnailForVideoInner(videoPath)
	if err != nil {
		cx.thumbnailPaths[id] = "/cache/error.jpg"
	} else {
		cx.thumbnailPaths[id] = path
	}
	return path, err
}

func (cx *Context) generateThumbnailForVideoInner(videoPath string) (string, error) {
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
	dirId := len(cx.directories)
	directory := Directory{id: dirId, childDirectory: childDirectory, files: files, name: name}
	cx.directories = append(cx.directories, directory)
	return directory, nil
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

func fullscreenUrl(path string, file File) string {
	prefix := "/fullscreen/"
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

func getFilesInRange(cx *Context, directory *Directory, path string, index int) []FileData {
	files := directory.files
	if len(files) <= 11 {
		return getFilesInRangeInner(cx, path, files)
	}
	start, end := getIndexRange(index)
	return getFilesInRangeInner(cx, path, files[start:end])
}

func getFilesInRangeInner(cx *Context, path string, files []File) []FileData {
	data := make([]FileData, 0)
	for _, file := range files {
		if file.kind == Other {
			continue
		}
		data = append(data, FileData{Name: file.name, Url: slideUrl(path, file), ResourceUrl: fileResourceUrl(file), ThumnailUrl: fileThumbnailUrl(cx, file)})
	}
	return data
}

func fileDataInner(cx *Context, directory *Directory, path string, limit int) []FileData {
	data := make([]FileData, 0)
	for _, file := range directory.files {
		if file.kind == Other {
			continue
		}
		data = append(data, FileData{Name: file.name, Url: slideUrl(path, file), ResourceUrl: fileResourceUrl(file), ThumnailUrl: fileThumbnailUrl(cx, file)})
		if len(data) == limit {
			break
		}
	}
	return data
}

func main() {
	cx := Context{paths: make([]string, 0), directories: make([]Directory, 0)}
	cx.cleanCache()
	dataPath := os.Args[1]
	dir, err := addDirRootToContext(&cx, dataPath)
	if err != nil {
		panic(err)
	}
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
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
			handleInvalidFile(c, err.Error())
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
			handleInvalidFile(c, err.Error())
			return
		}
		returnFileById(&cx, c, id)
	})
	r.GET("/image-grid/:directoryId", func(c *gin.Context) {
		directoryIdStr := c.Param("directoryId")
		directoryId, err := strconv.Atoi(directoryIdStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}
		returnImageGrid(&cx, c, directoryId)
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
			handleInvalidFile(c, err.Error())
			return
		}
		index := index(directory.files, id)
		if index == -1 {
			handleInvalidFile(c, "File not found")
			return
		}
		isVideo := directory.files[index].kind == Video
		prev := prevUrl(directory.files, index, path)
		next := nextUrl(directory.files, index, path)
		resourceUrl := imageResourceUrlById(id)
		c.HTML(http.StatusOK, "slide.tmpl", gin.H{
			"Name":        directory.files[index].name,
			"isVideo":     isVideo,
			"ResourceUrl": resourceUrl,
			"PrevUrl":     prev,
			"NextUrl":     next,
			"DirectoryId": directory.id,
			"FileId":      id,
			"Path":        path,
		})
	})

	r.GET("/fullscreen/:id/*path", func(c *gin.Context) {
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
			handleInvalidFile(c, err.Error())
			return
		}
		index := index(directory.files, id)
		if index == -1 {
			handleInvalidFile(c, "File not found")
			return
		}

		isVideo := directory.files[index].kind == Video
		prev := prevFullscreenUrl(directory.files, index, path)
		next := nextFullscreenUrl(directory.files, index, path)
		resourceUrl := imageResourceUrlById(id)

		// Count total images/videos for counter
		totalCount := 0
		for _, file := range directory.files {
			if file.kind == Image || file.kind == Video {
				totalCount++
			}
		}

		// Count current position (only images/videos)
		currentIndex := 1
		for i := 0; i < index; i++ {
			if directory.files[i].kind == Image || directory.files[i].kind == Video {
				currentIndex++
			}
		}

		c.HTML(http.StatusOK, "fullscreen.tmpl", gin.H{
			"Name":         directory.files[index].name,
			"IsVideo":      isVideo,
			"ResourceUrl":  resourceUrl,
			"PrevUrl":      prev,
			"NextUrl":      next,
			"CurrentIndex": currentIndex,
			"TotalCount":   totalCount,
			"IsFirst":      index == 0,
			"IsLast":       index == len(directory.files)-1,
		})
	})

	fmt.Println(dir)
	r.Run()
}

func getIndexRange(index int) (int, int) {
	start := index - 5
	if start < 0 {
		start = 0
	}
	end := start + 10
	return start, end
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

func nextFullscreenUrl(files []File, i int, path string) string {
	if i+1 < len(files) {
		return fullscreenUrl(path, files[i+1])
	} else {
		return fullscreenUrl(path, files[0])
	}
}

func prevFullscreenUrl(files []File, i int, path string) string {
	if i-1 >= 0 {
		return fullscreenUrl(path, files[i-1])
	} else {
		return fullscreenUrl(path, files[len(files)-1])
	}
}

func returnDirectoryPage(c *gin.Context, cx *Context, directory *Directory, path string) {
	Directories := childDirectoryData(directory, path)
	c.HTML(http.StatusOK, "directoryData.tmpl", gin.H{
		"name":        directory.name,
		"Directories": Directories,
		"DirectoryId": directory.id,
	})
}

func returnFileByPath(c *gin.Context, path string) {
	file, err := os.Open(path)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}

	fileSize := fileInfo.Size()
	buffer := make([]byte, fileSize)

	_, err = file.Read(buffer)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}

	contentType := http.DetectContentType(buffer)
	c.Data(http.StatusOK, contentType, buffer)
}

func returnFileById(cx *Context, c *gin.Context, id int) {
	path, err := cx.getPath(id)
	if err != nil {
		handleInvalidFile(c, err.Error())
		return
	}
	returnFileByPath(c, path)
}

func returnImageGrid(cx *Context, c *gin.Context, directoryId int) {
	directory, err := cx.getDirectoryById(directoryId)
	if err != nil {
		handleInvalidFile(c, err.Error())
		return
	}
	files := fileDataInner(cx, directory, "", 0) // Get all files, no limit
	c.HTML(http.StatusOK, "imageGrid.tmpl", gin.H{
		"Files": files,
	})
}

func handleError(c *gin.Context, statusCode int, reason string) {
	log.Printf("Error: %s", reason)
	c.HTML(statusCode, "error.tmpl", gin.H{
		"reason": reason,
	})
}

func handleInvalidFile(c *gin.Context, reason string) {
	log.Printf("Invalid file error: %s", reason)
	c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
		"reason": reason,
	})
}
