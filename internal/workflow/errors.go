package workflow

import "errors"

var (
	ErrNotFound       = errors.New("资源不存在")
	ErrConflict       = errors.New("资源已存在")
	ErrIdempotencyKey = errors.New("idempotencyKey 已被其他命令使用")
)
