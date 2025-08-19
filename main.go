package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type FileKind int

const (
	Image FileKind = iota
	Video
	Other
)

const (
	PageSize          = 10
	SlideshowInterval = 5000 // 5 seconds in milliseconds
)

type File struct {
	name     string
	id       int
	kind     FileKind
	dirPath  []int
	modTime  time.Time
	filePath string
	deleted  bool
}

type Directory struct {
	id             int
	name           string
	files          []int
	childDirectory []Directory
	modTime        time.Time
}

type Context struct {
	directories []Directory
	files       []File
	rootDir     *Directory
}

func (cx *Context) getDirectoryById(id int) (*Directory, error) {
	if id < 0 || id >= len(cx.directories) {
		return nil, fmt.Errorf("invalid directory id %d", id)
	}
	return &cx.directories[id], nil
}

func (cx *Context) getFileById(id int) (*File, error) {
	if id < 0 || id >= len(cx.files) {
		return nil, fmt.Errorf("invalid file id %d", id)
	}
	return &cx.files[id], nil
}

// buildFilePath constructs the full path to a file using its dirPath array with clickable links
func (cx *Context) buildFilePath(file File, sortParam string) string {
	if len(file.dirPath) == 0 {
		return file.name
	}

	pathParts := make([]string, 0, len(file.dirPath)+1)
	currentPath := ""

	// Build path parts with links for directories
	for _, dirId := range file.dirPath {
		if dirId == 0 {
			// Virtual root di
			continue
		}
		if dirId >= 0 && dirId < len(cx.directories) {
			dirName := cx.directories[dirId].name
			if currentPath == "" {
				currentPath = dirName
			} else {
				currentPath = currentPath + "/" + dirName
			}
			link := ""
			if sortParam != "" {
				link = fmt.Sprintf(`<a href="/files/%s?sort=%s">%s</a>`, currentPath, sortParam, dirName)
			} else {
				link = fmt.Sprintf(`<a href="/files/%s">%s</a>`, currentPath, dirName)
			}
			pathParts = append(pathParts, link)
		}
	}

	// Add the file name (not clickable)
	pathParts = append(pathParts, file.name)

	return strings.Join(pathParts, " / ")
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

func addFileToContext(cx *Context, path string, dirPath []int) (int, error) {
	log.Printf("addFileToContext: path=%s dirPath=%v", path, dirPath)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	name := fileInfo.Name()
	id := len(cx.files)
	file := File{name: name, id: id, kind: fileKind(path), dirPath: dirPath, modTime: fileInfo.ModTime(), filePath: path, deleted: false}
	cx.files = append(cx.files, file)
	return id, nil
}

func fileKind(path string) FileKind {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return Image
	case ".mp4", ".webm", ".ogg", ".ogv", ".mov":
		return Video
	default:
		return Other
	}
}

func filteredFile(path string) bool {
	return filepath.Base(path)[0] == '.'
}

func addDirRootToContext(cx *Context, path string, parentPath []int) (Directory, error) {
	dirId := len(cx.directories)
	currentPath := append(parentPath, dirId)
	log.Printf("addDirRootToContext: path=%s currentPath=%v", path, currentPath)

	name := filepath.Base(path)
	// Capture directory metadata
	dirInfo, err := os.Stat(path)
	if err != nil {
		return Directory{}, err
	}
	directory := Directory{id: dirId, name: name, files: []int{}, childDirectory: []Directory{}, modTime: dirInfo.ModTime()}
	cx.directories = append(cx.directories, directory)

	childDirectory := make([]Directory, 0)
	files := make([]int, 0)

	entries, err := os.ReadDir(path)
	if err != nil {
		return Directory{}, err
	}

	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())
		if filteredFile(childPath) {
			continue
		}
		if entry.IsDir() {
			// Recurse into child directories, but only add their metadata here
			dir, err := addDirRootToContext(cx, childPath, currentPath)
			if err != nil {
				return Directory{}, err
			}
			childDirectory = append(childDirectory, dir)
		} else {
			file, err := addFileToContext(cx, childPath, currentPath)
			if err != nil {
				return Directory{}, err
			}
			files = append(files, file)
		}
	}

	cx.directories[dirId].childDirectory = childDirectory
	cx.directories[dirId].files = files

	log.Printf("scanned directory: name=%s id=%d files=%d children=%d", name, dirId, len(files), len(childDirectory))

	return cx.directories[dirId], nil
}

type DirectoryData struct {
	Name string
	Url  string
}

type FileData struct {
	Name         string
	Url          string
	ResourceUrl  string
	ThumbnailUrl string
	IsVideo      bool
}

func directoryUrl(path, name string) string {
	basePath := strings.Trim(path, "/")
	prefix := "/files/"
	if basePath == "" {
		return prefix + name
	}
	return prefix + basePath + "/" + name
}

func directoryUrlWithSort(path, name, sortParam string) string {
	url := directoryUrl(path, name)
	if sortParam == "" {
		return url
	}
	return url + "?sort=" + sortParam
}

func childDirectoryData(directory *Directory, path string, sortParam string) []DirectoryData {
	// copy to avoid mutating original slice order
	dirs := make([]Directory, len(directory.childDirectory))
	copy(dirs, directory.childDirectory)
	// sort according to sortParam
	switch strings.ToLower(sortParam) {
	case "latest":
		sort.SliceStable(dirs, func(i, j int) bool { return dirs[i].modTime.After(dirs[j].modTime) })
	case "oldest":
		sort.SliceStable(dirs, func(i, j int) bool { return dirs[i].modTime.Before(dirs[j].modTime) })
	default: // name
		sort.SliceStable(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name) })
	}

	data := make([]DirectoryData, 0, len(dirs))
	for _, dir := range dirs {
		data = append(data, DirectoryData{Name: dir.name, Url: directoryUrlWithSort(path, dir.name, sortParam)})
	}
	return data
}

func slideUrl(path string, file File, sortParam string) string {
	prefix := "/slides/"
	basePath := strings.Trim(path, "/")
	id := strconv.Itoa(file.id)
	var url string
	if basePath == "" {
		url = prefix + id
	} else {
		url = prefix + id + "/" + basePath
	}
	if sortParam != "" {
		url = url + "?sort=" + sortParam
	}
	return url
}

func fullscreenUrl(path string, file File, sortParam string) string {
	prefix := "/fullscreen/"
	basePath := strings.Trim(path, "/")
	id := strconv.Itoa(file.id)
	var url string
	if basePath == "" {
		url = prefix + id
	} else {
		url = prefix + id + "/" + basePath
	}
	if sortParam != "" {
		url = url + "?sort=" + sortParam
	}
	return url
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
		// For videos, use the video file itself as thumbnail
		// The browser will display the first frame
		// FIXME: for this to work we need to use a video tag not an img tag
		return videoResourceUrlById(file.id)
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

func getSortedMediaFiles(cx *Context, files []int, sortParam string) []File {
	mediaFiles := make([]File, 0)
	for _, idx := range files {
		f := cx.files[idx]
		if f.kind != Other && !f.deleted {
			mediaFiles = append(mediaFiles, f)
		}
	}
	switch strings.ToLower(sortParam) {
	case "latest":
		sort.SliceStable(mediaFiles, func(i, j int) bool { return mediaFiles[i].modTime.After(mediaFiles[j].modTime) })
	case "oldest":
		sort.SliceStable(mediaFiles, func(i, j int) bool { return mediaFiles[i].modTime.Before(mediaFiles[j].modTime) })
	default: // name
		sort.SliceStable(mediaFiles, func(i, j int) bool { return strings.ToLower(mediaFiles[i].name) < strings.ToLower(mediaFiles[j].name) })
	}
	return mediaFiles
}


func fileDataInRange(cx *Context, directory *Directory, path string, start int, end int, sortParam string) []FileData {
	data := make([]FileData, 0)
	mediaFiles := getSortedMediaFiles(cx, directory.files, sortParam)

	totalFiles := len(mediaFiles)
	if totalFiles == 0 {
		return data
	}

	// Handle wrap-around by normalizing indices (including negative ones)
	for i := start; i < end; i++ {
		index := ((i % totalFiles) + totalFiles) % totalFiles
		file := mediaFiles[index]

		data = append(data, FileData{
			Name:         file.name,
			Url:          slideUrl(path, file, sortParam),
			ResourceUrl:  fileResourceUrl(file),
			ThumbnailUrl: fileThumbnailUrl(cx, file),
			IsVideo:      file.kind == Video,
		})
	}

	return data
}

func countMediaFiles(cx *Context, directory *Directory) int {
	count := 0
	for _, idx := range directory.files {
		file := cx.files[idx]
		if file.kind != Other && !file.deleted {
			count++
		}
	}
	return count
}

// findNextNonDeletedById returns the file with the smallest id greater than afterId;
// if none, it returns the file with the smallest id. Only considers non-deleted media files.
func findNextNonDeletedById(cx *Context, files []int, afterId int) (File, bool) {
	var (
		nextCandidate File
		foundNext     bool
		smallest      File
		foundSmallest bool
	)
	for _, id := range files {
		f := cx.files[id]
		if f.kind == Other || f.deleted {
			continue
		}
		if !foundSmallest || f.id < smallest.id {
			smallest = f
			foundSmallest = true
		}
		if f.id > afterId {
			if !foundNext || f.id < nextCandidate.id {
				nextCandidate = f
				foundNext = true
			}
		}
	}
	if foundNext {
		return nextCandidate, true
	}
	if foundSmallest {
		return smallest, true
	}
	return File{}, false
}

func validateContext(cx *Context) {
	for i, file := range cx.files {
		if file.id != i {
			fmt.Println("invalid index")
			os.Exit(1)
		}
	}
}

func main() {
	cx := Context{directories: make([]Directory, 0), files: make([]File, 0)}

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var rootDir Directory

	if len(config.MediaSources) == 1 {
		// Single media source - use it directly
		dir, err := addDirRootToContext(&cx, config.MediaSources[0], []int{})
		if err != nil {
			panic(err)
		}
		rootDir = dir
	} else {
		// Multiple media sources - create virtual directory
		directories := make([]Directory, 0, len(config.MediaSources))
		virtualDir := Directory{
			id:             0,
			name:           "$Virtual",
			files:          []int{},
			childDirectory: directories,
		}
		cx.directories = append(cx.directories, virtualDir)
		for _, path := range config.MediaSources {
			dir, err := addDirRootToContext(&cx, path, []int{0})
			if err != nil {
				log.Printf("Warning: failed to add directory %s: %v", path, err)
				continue
			}
			directories = append(directories, dir)
		}

		if len(directories) == 0 {
			log.Fatal("No valid media sources found")
		}
		rootDir = combineDirectories(&cx, directories)
	}
	cx.rootDir = &rootDir
	validateContext(&cx)
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		returnDirectoryPage(c, &cx, &rootDir, "")
	})
	r.GET("/files/*path", func(c *gin.Context) {
		path := c.Param("path")
		directory := getDirectoryByPath(&rootDir, path)
		if directory == nil {
			c.HTML(http.StatusNotFound, "invalidPath.tmpl", gin.H{
				"path": path,
			})
			return
		}
		returnDirectoryPage(c, &cx, directory, path)
	})
	r.GET("/img/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}
		returnFileById(&cx, c, id)
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
	r.GET("/image-grid/:directoryId/*path", func(c *gin.Context) {
		directoryIdStr := c.Param("directoryId")
		directoryId, err := strconv.Atoi(directoryIdStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}

		startStr := c.DefaultQuery("start", "0")
		start, err := strconv.Atoi(startStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}

		endStr := c.DefaultQuery("end", "10")
		end, err := strconv.Atoi(endStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}

		path := c.Param("path")
		sortParam := strings.ToLower(c.DefaultQuery("sort", "name"))
		returnImageGrid(&cx, c, directoryId, start, end, path, sortParam)
	})

	r.GET("/slides/:id/*path", func(c *gin.Context) {
		req, err := parseSlideReq(c)
		if err != nil {
			handleError(c, 400, err.Error())
		}
		res, err := createParseRes(&cx, req)
		if err != nil {
			handleError(c, 500, err.Error())
			return
		}
		c.HTML(http.StatusOK, "slide.tmpl", gin.H{
			"Name":        res.name,
			"FilePath":    template.HTML(res.filePath),
			"isVideo":     res.isVideo,
			"ResourceUrl": res.resourceUrl,
			"PrevUrl":     res.prevUrl,
			"NextUrl":     res.nextUrl,
			"DirectoryId": res.directoryId,
			"FileId":      res.fileId,
			"Path":        res.path,
			"GridStart":   res.gridStart,
			"GridEnd":     res.gridEnd,
			"Sort":        req.sortBy.toStr(),
		})
	})

	r.GET("/fullscreen/:id/*path", func(c *gin.Context) {
		path := c.Param("path")
		sortParam := strings.ToLower(c.DefaultQuery("sort", "name"))
		idStr := c.Param("id")
		directory := getDirectoryByPath(&rootDir, path)
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
		sortedFiles := getSortedMediaFiles(&cx, directory.files, sortParam)
		index := index(sortedFiles, id)
		if index == -1 {
			if nextFile, ok := findNextNonDeletedById(&cx, directory.files, id); ok {
				c.Redirect(http.StatusSeeOther, fullscreenUrl(path, nextFile, sortParam))
				return
			}
			redir := "/files" + path
			if sortParam != "" {
				redir = redir + "?sort=" + sortParam
			}
			c.Redirect(http.StatusSeeOther, redir)
			return
		}

		isVideo := sortedFiles[index].kind == Video
		prev := prevFullscreenUrl(sortedFiles, index, path, sortParam)
		next := nextFullscreenUrl(sortedFiles, index, path, sortParam)
		resourceUrl := fileResourceUrl(sortedFiles[index])

		// Count total images/videos for counter based on sorted files
		totalCount := len(sortedFiles)

		// Current position (1-based)
		currentIndex := index + 1

		c.HTML(http.StatusOK, "fullscreen.tmpl", gin.H{
			"Name":              sortedFiles[index].name,
			"IsVideo":           isVideo,
			"ResourceUrl":       resourceUrl,
			"PrevUrl":           prev,
			"NextUrl":           next,
			"CurrentIndex":      currentIndex,
			"TotalCount":        totalCount,
			"IsFirst":           index == 0,
			"IsLast":            index == len(sortedFiles)-1,
			"SlideshowInterval": SlideshowInterval,
			"Sort":              sortParam,
		})
	})

	// Delete endpoint: move file to system trash, mark tombstone, redirect
	r.POST("/delete", func(c *gin.Context) {
		req, err := parseDeleteReq(c)
		pathParam := c.PostForm("path")
		sortParam := strings.ToLower(c.DefaultPostForm("sort", "name"))
		if err != nil {
			handleError(c, 400, err.Error())
			return
		}
		nextFile, err := handleDelete(&cx, c, *req)
		if err != nil {
			handleError(c, 500, err.Error())
			return
		}
		redirectUrl := slideUrl(pathParam, *nextFile, sortParam)
		c.Redirect(http.StatusSeeOther, redirectUrl)
	})

	r.Run(fmt.Sprintf(":%d", config.Port))
}

func index(files []File, id int) int {
	for i, file := range files {
		if file.id == id {
			return i
		}
	}
	return -1
}

func nextUrl(files []File, i int, path string, sortParam string) string {
	if i+1 < len(files) {
		return slideUrl(path, files[i+1], sortParam)
	} else {
		return slideUrl(path, files[0], sortParam)
	}
}

func prevUrl(files []File, i int, path string, sortParam string) string {
	if i-1 >= 0 {
		return slideUrl(path, files[i-1], sortParam)
	} else {
		return slideUrl(path, files[len(files)-1], sortParam)
	}
}

func nextFullscreenUrl(files []File, i int, path string, sortParam string) string {
	if i+1 < len(files) {
		return fullscreenUrl(path, files[i+1], sortParam)
	} else {
		return fullscreenUrl(path, files[0], sortParam)
	}
}

func prevFullscreenUrl(files []File, i int, path string, sortParam string) string {
	if i-1 >= 0 {
		return fullscreenUrl(path, files[i-1], sortParam)
	} else {
		return fullscreenUrl(path, files[len(files)-1], sortParam)
	}
}

func returnDirectoryPage(c *gin.Context, cx *Context, directory *Directory, path string) {
	sortParam := strings.ToLower(c.DefaultQuery("sort", "name"))
	Directories := childDirectoryData(directory, path, sortParam)
	hasDirectories := len(Directories) > 0
	hasFiles := countMediaFiles(cx, directory) > 0
	c.HTML(http.StatusOK, "directoryData.tmpl", gin.H{
		"name":           directory.name,
		"Directories":    Directories,
		"DirectoryId":    directory.id,
		"Path":           path,
		"Sort":           sortParam,
		"HasDirectories": hasDirectories,
		"HasFiles":       hasFiles,
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

	// Use http.ServeContent for better HTTP serving with range requests support
	http.ServeContent(c.Writer, c.Request, fileInfo.Name(), fileInfo.ModTime(), file)
}

func returnFileById(cx *Context, c *gin.Context, id int) {
	// If file tombstoned, treat as invalid
	file, err := cx.getFileById(id)
	if err != nil {
		handleInvalidFile(c, fmt.Sprintf("invalid id %d", id))
		return
	}
	if file.deleted {
		handleInvalidFile(c, "File deleted")
		return
	}
	path := file.filePath
	returnFileByPath(c, path)
}

func returnImageGrid(cx *Context, c *gin.Context, directoryId int, start int, end int, path string, sortParam string) {
	directory, err := cx.getDirectoryById(directoryId)
	if err != nil {
		handleInvalidFile(c, err.Error())
		return
	}
	files := fileDataInRange(cx, directory, path, start, end, sortParam)
	nextStart := end
	nextEnd := end + PageSize
	totalFiles := countMediaFiles(cx, directory)

	// Always show more button since we have wrap-around
	// Only hide if there are no files at all
	hasMore := totalFiles > 0

	c.HTML(http.StatusOK, "imageGrid.tmpl", gin.H{
		"Files":       files,
		"HasMore":     hasMore,
		"NextStart":   nextStart,
		"NextEnd":     nextEnd,
		"DirectoryId": directoryId,
		"Path":        path,
		"Sort":        sortParam,
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

// combineDirectories creates a virtual directory that contains all the provided directories as child directories
func combineDirectories(cx *Context, directories []Directory) Directory {
	// Create a virtual root directory
	virtualDir := Directory{
		id:             len(cx.directories),
		name:           "Collections",
		files:          []int{},
		childDirectory: directories,
	}

	// Add the virtual directory to the context
	cx.directories = append(cx.directories, virtualDir)

	return virtualDir
}

type SortBy int

const (
	Name SortBy = iota
	Latest
	Oldest
)

type ReqBase struct {
	sortBy SortBy
}

func sortByFromStr(value string) (SortBy, error) {
	switch strings.ToLower(value) {
	case "latest":
		return Latest, nil
	case "oldest":
		return Oldest, nil
	case "name":
		return Name, nil
	default:
		return -1, errors.New("invalid sortBy value: " + value)
	}
}

func (s *SortBy) toStr() string {
	switch *s {
	case Latest:
		return "latest"
	case Oldest:
		return "oldest"
	default:
		return "name"
	}
}
