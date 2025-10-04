package main

import (
	"fmt"
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
		return nil, fmt.Errorf("trying to delete already deleted file at %d", file.id)
	}
	dir, err := cx.getDirectoryById(req.dirId)
	if err != nil {
		return nil, err
	}

	// We need to get files before marking file as deleted otherwise it will not be in the list
	filesInDir := getSortedMediaFiles(cx, dir.files, req.sortBy.toStr())
	file.deleted = true
	nextId := -1
	for i, f := range filesInDir {
		if file.id == f.id {
			nextId = (i + 1) % len(filesInDir)
			break
		}
	}
	if nextId == -1 {
		return nil, fmt.Errorf("failed to find file in directory %d", dir.id)
	}

	path := file.filePath
	err = cx.systemUtils.MoveToTrash(path)
	if err != nil {
		return nil, fmt.Errorf("failed to delete file %d due to %s", file.id, err.Error())
	}
	return &filesInDir[nextId], nil
}
