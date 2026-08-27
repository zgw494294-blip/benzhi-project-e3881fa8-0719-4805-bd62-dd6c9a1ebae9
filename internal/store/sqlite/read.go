package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/workflow"
)

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return workflow.ErrNotFound
	}
	return err
}

func (r *Repository) LoadPackage(ctx context.Context, packageID string) (caption.CaptionPackage, error) {
	var body []byte
	err := r.db.QueryRowContext(ctx, `SELECT body FROM packages WHERE package_id = ?`, packageID).Scan(&body)
	if err != nil {
		return caption.CaptionPackage{}, notFound(err)
	}
	var value caption.CaptionPackage
	return value, decode(body, &value)
}

func (r *Repository) ListPackages(ctx context.Context) ([]caption.CaptionPackage, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT body FROM packages ORDER BY updated_at DESC, package_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]caption.CaptionPackage, 0)
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		var value caption.CaptionPackage
		if err := decode(body, &value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *Repository) LoadRevision(ctx context.Context, revisionID string) (caption.CaptionRevision, error) {
	var body []byte
	err := r.db.QueryRowContext(ctx, `SELECT body FROM revisions WHERE revision_id = ?`, revisionID).Scan(&body)
	if err != nil {
		return caption.CaptionRevision{}, notFound(err)
	}
	var value caption.CaptionRevision
	return value, decode(body, &value)
}

func (r *Repository) ListRevisions(ctx context.Context, packageID string) ([]caption.CaptionRevision, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT body FROM revisions WHERE package_id = ? ORDER BY submitted_at, revision_id`, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]caption.CaptionRevision, 0)
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		var value caption.CaptionRevision
		if err := decode(body, &value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *Repository) LoadFindings(ctx context.Context, revisionID string) ([]caption.QualityFinding, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT body FROM findings WHERE revision_id = ? ORDER BY finding_id`, revisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]caption.QualityFinding, 0)
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		var value caption.QualityFinding
		if err := decode(body, &value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *Repository) LoadSummary(ctx context.Context, revisionID string) (caption.CheckSummary, error) {
	var body []byte
	err := r.db.QueryRowContext(ctx, `SELECT body FROM check_summaries WHERE revision_id = ?`, revisionID).Scan(&body)
	if err != nil {
		return caption.CheckSummary{}, notFound(err)
	}
	var value caption.CheckSummary
	return value, decode(body, &value)
}

func (r *Repository) LoadLatestReview(ctx context.Context, packageID string) (caption.ReviewDecision, error) {
	var body []byte
	err := r.db.QueryRowContext(ctx, `SELECT body FROM reviews WHERE package_id = ? ORDER BY decided_at DESC, decision_id DESC LIMIT 1`, packageID).Scan(&body)
	if err != nil {
		return caption.ReviewDecision{}, notFound(err)
	}
	var value caption.ReviewDecision
	return value, decode(body, &value)
}

func (r *Repository) LoadManifest(ctx context.Context, packageID string) (caption.ReleaseManifest, error) {
	var body []byte
	err := r.db.QueryRowContext(ctx, `SELECT body FROM manifests WHERE package_id = ? ORDER BY issued_at DESC, manifest_id DESC LIMIT 1`, packageID).Scan(&body)
	if err != nil {
		return caption.ReleaseManifest{}, notFound(err)
	}
	var value caption.ReleaseManifest
	return value, decode(body, &value)
}

func (r *Repository) LoadEvents(ctx context.Context, packageID string) ([]audit.Event, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT body FROM audit_events WHERE package_id = ? ORDER BY sequence`, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]audit.Event, 0)
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		var value audit.Event
		if err := decode(body, &value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *Repository) LoadIdempotency(ctx context.Context, key string) (workflow.IdempotencyRecord, error) {
	var value workflow.IdempotencyRecord
	err := r.db.QueryRowContext(ctx, `SELECT key, command, package_id, response FROM idempotency WHERE key = ?`, key).Scan(&value.Key, &value.Command, &value.PackageID, &value.Response)
	if err != nil {
		return workflow.IdempotencyRecord{}, notFound(err)
	}
	if value.Key == "" {
		return workflow.IdempotencyRecord{}, fmt.Errorf("无效幂等记录")
	}
	return value, nil
}
