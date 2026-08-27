package caption

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidInput        = errors.New("输入数据不合法")
	ErrInvalidState        = errors.New("当前状态不允许该操作")
	ErrVersionConflict     = errors.New("版本冲突")
	ErrDuplicateCue        = errors.New("cueId 重复")
	ErrActorNotIndependent = errors.New("复核员必须与修订提交者不同")
	ErrBlockingFindings    = errors.New("仍有未解决的阻断问题")
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("%v: %s", ErrInvalidInput, e.Message)
	}
	return fmt.Sprintf("%v: %s: %s", ErrInvalidInput, e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error { return ErrInvalidInput }

func Invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
