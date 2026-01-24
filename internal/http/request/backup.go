package request

import "mime/multipart"

type BackupList struct {
	Type string `uri:"type" form:"type" validate:"required|in:path,website,mysql,postgres,redis,panel"`
}

type BackupCreate struct {
	Type    string `uri:"type" form:"type" validate:"required|in:website,mysql,postgres,redis,panel"`
	Target  string `json:"target" form:"target" validate:"required|regex:^[a-zA-Z0-9_-]+$"`
	Storage uint   `form:"storage" json:"storage"`
}

type BackupUpload struct {
	Type string                `uri:"type" form:"type"` // 校验没有必要，因为根本没经过验证器
	File *multipart.FileHeader `form:"file"`
}

type BackupFile struct {
	Type string `uri:"type" form:"type" validate:"required|in:website,mysql,postgres,redis,panel"`
	File string `json:"file" form:"file" validate:"required"`
}

type BackupRestore struct {
	BackupFile
	Target string `json:"target" form:"target" validate:"required|regex:^[a-zA-Z0-9_-]+$"`
}
