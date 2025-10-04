package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type slideReq struct {
	ReqBase
	fileId int
	path   string
}

type slideRes struct {
	name        string
	filePath    string
	isVideo     bool
	resourceUrl string
	prevUrl     string
	nextUrl     string
	directoryId int
	fileId      int
	path        string
	gridStart   int
	gridEnd     int
}

func parseSlideReq(c *gin.Context) (*slideReq, error) {
	path := c.Param("path")
	sortParam := strings.ToLower(c.DefaultQuery("sort", "name"))
	sortBy, err := sortByFromStr(sortParam)
	if err != nil {
		return nil, err
	}
	idStr := c.Param("id")
	fileId, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, err
	}
	return &slideReq{ReqBase{sortBy}, fileId, path}, nil
}

func createParseRes(cx *Context, req *slideReq) (*slideRes, error) {
	directory := getDirectoryByPath(cx.rootDir, req.path)
	if directory == nil {
		return nil, fmt.Errorf("failed to find directory matching path: %s", req.path)
	}
	sortedFiles := getSortedMediaFiles(cx, directory.files, req.sortBy.toStr())

	// Check if directory is empty
	if len(sortedFiles) == 0 {
		return nil, fmt.Errorf("directory is empty: %s", req.path)
	}

	index := index(sortedFiles, req.fileId)
	if index == -1 {
		return nil, fmt.Errorf("file with id %d not found in directory: %s", req.fileId, req.path)
	}

	isVideo := sortedFiles[index].kind == Video
	prev := prevUrl(sortedFiles, index, req.path, req.sortBy.toStr())
	next := nextUrl(sortedFiles, index, req.path, req.sortBy.toStr())
	resourceUrl := fileResourceURL(sortedFiles[index])

	currentPosition := index
	totalFiles := len(sortedFiles)
	gridStart := currentPosition - PageSize/2
	gridEnd := currentPosition + PageSize/2

	// Handle negative wrap-around
	if gridStart < 0 && totalFiles > 0 {
		gridStart = totalFiles + gridStart
	}

	filePath := cx.buildFilePath(sortedFiles[index], req.sortBy.toStr())
	return &slideRes{
		sortedFiles[index].name,
		filePath,
		isVideo,
		resourceUrl,
		prev,
		next,
		directory.id,
		req.fileId,
		req.path,
		gridStart,
		gridEnd,
	}, nil
}
