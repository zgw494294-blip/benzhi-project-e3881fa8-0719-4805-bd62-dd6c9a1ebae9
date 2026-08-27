package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

const EventChainVersion = "audit-chain/1"

type Event struct {
	EventID        string          `json:"eventId"`
	PackageID      string          `json:"packageId"`
	Sequence       int64           `json:"sequence"`
	Type           string          `json:"type"`
	ActorID        string          `json:"actorId"`
	OccurredAt     time.Time       `json:"occurredAt"`
	Data           json.RawMessage `json:"data"`
	PreviousDigest string          `json:"previousDigest"`
	Digest         string          `json:"digest"`
}

func NewEvent(packageID string, sequence int64, eventType, actorID string, at time.Time, data any, previousDigest string) (Event, error) {
	if packageID == "" || eventType == "" || actorID == "" {
		return Event{}, fmt.Errorf("审计事件缺少必填字段")
	}
	if sequence <= 0 {
		return Event{}, fmt.Errorf("审计事件序号必须为正数")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return Event{}, fmt.Errorf("编码审计事件: %w", err)
	}
	event := Event{PackageID: packageID, Sequence: sequence, Type: eventType, ActorID: actorID, OccurredAt: at.UTC(), Data: payload, PreviousDigest: previousDigest}
	event.EventID = fmt.Sprintf("%s-%06d", packageID, sequence)
	event.Digest = eventDigest(event)
	return event, nil
}

func eventDigest(event Event) string {
	canonical := struct {
		Version        string          `json:"version"`
		PackageID      string          `json:"packageId"`
		Sequence       int64           `json:"sequence"`
		Type           string          `json:"type"`
		ActorID        string          `json:"actorId"`
		OccurredAt     string          `json:"occurredAt"`
		Data           json.RawMessage `json:"data"`
		PreviousDigest string          `json:"previousDigest"`
	}{EventChainVersion, event.PackageID, event.Sequence, event.Type, event.ActorID, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.Data, event.PreviousDigest}
	encoded, _ := json.Marshal(canonical)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func VerifyChain(events []Event) error {
	previous := ""
	for index, event := range events {
		expectedSequence := int64(index + 1)
		if event.Sequence != expectedSequence {
			return fmt.Errorf("审计事件序号不连续: 期望 %d，实际 %d", expectedSequence, event.Sequence)
		}
		if event.PreviousDigest != previous {
			return fmt.Errorf("审计事件 %d 的前序摘要不匹配", event.Sequence)
		}
		if eventDigest(event) != event.Digest {
			return fmt.Errorf("审计事件 %d 内容摘要损坏", event.Sequence)
		}
		previous = event.Digest
	}
	return nil
}

func NextLink(events []Event) (int64, string, error) {
	if err := VerifyChain(events); err != nil {
		return 0, "", err
	}
	if len(events) == 0 {
		return 1, "", nil
	}
	last := events[len(events)-1]
	return last.Sequence + 1, last.Digest, nil
}
