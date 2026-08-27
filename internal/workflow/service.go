package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
)

type Service struct {
	repo         Repository
	audit        *audit.Service
	clock        Clock
	historyMu    sync.RWMutex
	historyCache map[string][]audit.Event
}

func NewService(repo Repository, auditService *audit.Service) *Service {
	return &Service{
		repo:         repo,
		audit:        auditService,
		clock:        func() time.Time { return time.Now().UTC() },
		historyCache: make(map[string][]audit.Event),
	}
}

func (s *Service) SetClock(clock Clock) {
	if clock != nil {
		s.clock = clock
	}
}
func (s *Service) Close() error { return s.repo.Close() }

func newID(prefix string) string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(bytes)
}

func validateCommand(packageID, actorID, key string) error {
	if strings.TrimSpace(packageID) == "" {
		return caption.Invalid("packageId", "不能为空")
	}
	if strings.TrimSpace(actorID) == "" {
		return caption.Invalid("actorId", "不能为空")
	}
	if len(strings.TrimSpace(key)) < 8 {
		return caption.Invalid("idempotencyKey", "至少需要 8 个字符")
	}
	return nil
}

func checkVersion(pkg caption.CaptionPackage, expected int64) error {
	if expected <= 0 {
		return caption.Invalid("expectedVersion", "必须为正数")
	}
	if pkg.Version != expected {
		return caption.ErrVersionConflict
	}
	return nil
}

func marshalResponse(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("编码幂等响应: %w", err)
	}
	return encoded, nil
}

func (s *Service) replay(ctx context.Context, key, command string, target any) (bool, error) {
	if strings.TrimSpace(key) == "" {
		return false, caption.Invalid("idempotencyKey", "不能为空")
	}
	record, err := s.repo.LoadIdempotency(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if record.Command != command {
		return false, ErrIdempotencyKey
	}
	if err := json.Unmarshal(record.Response, target); err != nil {
		return false, fmt.Errorf("读取幂等响应: %w", err)
	}
	return true, nil
}

func (s *Service) event(ctx context.Context, packageID, eventType, actor string, data any) (audit.Event, error) {
	history, err := s.repo.LoadEvents(ctx, packageID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return audit.Event{}, err
	}
	return s.audit.BuildEvent(packageID, eventType, actor, s.clock(), data, history)
}

func record(key, command, packageID string, response any) (IdempotencyRecord, error) {
	encoded, err := marshalResponse(response)
	if err != nil {
		return IdempotencyRecord{}, err
	}
	return IdempotencyRecord{Key: key, Command: command, PackageID: packageID, Response: encoded}, nil
}

func cloneEvents(events []audit.Event) []audit.Event {
	result := make([]audit.Event, len(events))
	for index, event := range events {
		result[index] = event
		result[index].Data = append([]byte(nil), event.Data...)
	}
	return result
}
