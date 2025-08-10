package main

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type deleteReq struct {
	ReqBase
	fileId int
	dirId  int
}

func parseDeleteReq(c *gin.Context) (*deleteReq, error) {
	fileIdStr := c.PostForm("fileId")
	dirIdStr := c.PostForm("directoryId")
	sortParam := strings.ToLower(c.DefaultPostForm("sort", "name"))

	fileId, err := strconv.Atoi(fileIdStr)
	if err != nil {
		return nil, err
	}
	dirId, err := strconv.Atoi(dirIdStr)
	if err != nil {
		return nil, err
	}

	sortBy, err := sortByFromStr(sortParam)
	if err != nil {
		return nil, err
	}
	return &deleteReq{ReqBase{sortBy}, fileId, dirId}, nil
}

func handleDelete(cx *Context, c *gin.Context, req deleteReq) (*File, error) {
	file, err := cx.getFileById(req.fileId)
	if err != nil {
		return nil, err
	}
	if file.deleted {
		return nil, fmt.Errorf("Trying to delete already deleted file at %d", file.id)
	}
	dir, err := cx.getDirectoryById(req.dirId)
	if err != nil {
		return nil, err
	}

	// We need to get files before marking file as deleted otherwise it will not be in the list
	filesInDir := getSortedMediaFilesNew(cx, dir.files, req.sortBy)
	file.deleted = true
	nextId := -1
	for i, f := range filesInDir {
		if file.id == f.id {
			nextId = (i + 1) % len(filesInDir)
			break
		}
	}
	if nextId == -1 {
		return nil, fmt.Errorf("Failed to find file in directory %d", dir.id)
	}

	path := file.filePath
	err = moveToTrash(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to delete file %d due to %s", file.id, err.Error())
	}
	return &filesInDir[nextId], nil
}

// moveToTrash attempts to move the file to the system's recycle bin.
// On Linux, it tries common mechanisms in order.
func moveToTrash(path string) error {
	// Prefer gio (GLib) trash which adheres to the FreeDesktop Trash spec
	if err := tryExec("gio", "trash", path); err == nil {
		return nil
	}
	// Older gvfs-trash
	if err := tryExec("gvfs-trash", path); err == nil {
		return nil
	}
	// trash-put from trash-cli
	if err := tryExec("trash-put", path); err == nil {
		return nil
	}
	// KDE kioclient5
	if err := tryExec("kioclient5", "move", path, "trash:/"); err == nil {
		return nil
	}
	return fmt.Errorf("no trash utility found (tried gio, gvfs-trash, trash-put, kioclient5)")
}

func tryExec(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func getSortedMediaFilesNew(cx *Context, files []int, sortBy SortBy) []File {
	mediaFiles := make([]File, 0)
	for _, idx := range files {
		f := cx.files[idx]
		if f.kind != Other {
			mediaFiles = append(mediaFiles, f)
		}
	}
	switch sortBy {
	case Latest:
		sort.SliceStable(mediaFiles, func(i, j int) bool { return mediaFiles[i].modTime.After(mediaFiles[j].modTime) })
	case Oldest:
		sort.SliceStable(mediaFiles, func(i, j int) bool { return mediaFiles[i].modTime.Before(mediaFiles[j].modTime) })
	default: // name
		sort.SliceStable(mediaFiles, func(i, j int) bool { return strings.ToLower(mediaFiles[i].name) < strings.ToLower(mediaFiles[j].name) })
	}
	return mediaFiles
}
