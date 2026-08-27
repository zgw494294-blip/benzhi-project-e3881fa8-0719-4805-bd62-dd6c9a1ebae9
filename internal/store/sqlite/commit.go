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

func (r *Repository) Commit(ctx context.Context, commit workflow.Commit) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := validateCommit(ctx, tx, commit); err != nil {
		return err
	}
	if err := savePackage(ctx, tx, commit.Package, commit.CreatePackage); err != nil {
		return err
	}
	if commit.Revision != nil {
		if err := insertRevision(ctx, tx, *commit.Revision); err != nil {
			return err
		}
	}
	if commit.ReplaceFindings {
		if err := replaceFindings(ctx, tx, commit); err != nil {
			return err
		}
	}
	if commit.Summary != nil {
		if err := upsertSummary(ctx, tx, *commit.Summary); err != nil {
			return err
		}
	}
	if commit.Review != nil {
		if err := insertReview(ctx, tx, *commit.Review); err != nil {
			return err
		}
	}
	if commit.Manifest != nil {
		if err := upsertManifest(ctx, tx, *commit.Manifest); err != nil {
			return err
		}
	}
	if err := insertEvent(ctx, tx, commit.Event); err != nil {
		return err
	}
	if err := insertIdempotency(ctx, tx, commit.Idempotency); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交 SQLite 事务: %w", err)
	}
	return nil
}

func validateCommit(ctx context.Context, tx *sql.Tx, commit workflow.Commit) error {
	if commit.Package.PackageID == "" || commit.Event.PackageID != commit.Package.PackageID {
		return fmt.Errorf("提交内容的 packageId 不一致")
	}
	if commit.Idempotency.PackageID != commit.Package.PackageID || commit.Idempotency.Key == "" {
		return fmt.Errorf("提交内容缺少幂等记录")
	}
	var currentVersion int64
	err := tx.QueryRowContext(ctx, `SELECT version FROM packages WHERE package_id = ?`, commit.Package.PackageID).Scan(&currentVersion)
	if commit.CreatePackage {
		if err == nil {
			return workflow.ErrConflict
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if commit.Package.Version != 1 || commit.Event.Sequence != 1 {
			return fmt.Errorf("新建包的版本或审计序号无效")
		}
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return workflow.ErrNotFound
	}
	if err != nil {
		return err
	}
	if currentVersion != commit.ExpectedVersion {
		return caption.ErrVersionConflict
	}
	if commit.Package.Version != currentVersion+1 {
		return fmt.Errorf("新版本必须递增 1")
	}
	var sequence sql.NullInt64
	var digest sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT sequence, digest FROM audit_events WHERE package_id = ? ORDER BY sequence DESC LIMIT 1`, commit.Package.PackageID).Scan(&sequence, &digest); err != nil {
		return err
	}
	expectedSequence := int64(1)
	expectedPrevious := ""
	if sequence.Valid {
		expectedSequence = sequence.Int64 + 1
		expectedPrevious = digest.String
	}
	if commit.Event.Sequence != expectedSequence || commit.Event.PreviousDigest != expectedPrevious {
		return fmt.Errorf("审计事件与数据库链尾不一致")
	}
	return nil
}

func savePackage(ctx context.Context, tx *sql.Tx, pkg caption.CaptionPackage, create bool) error {
	body, err := encode(pkg)
	if err != nil {
		return err
	}
	if create {
		_, err = tx.ExecContext(ctx, `INSERT INTO packages(package_id, version, status, updated_at, body) VALUES(?, ?, ?, ?, ?)`, pkg.PackageID, pkg.Version, pkg.Status, pkg.UpdatedAt.Format(timeFormat), body)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE packages SET version = ?, status = ?, updated_at = ?, body = ? WHERE package_id = ?`, pkg.Version, pkg.Status, pkg.UpdatedAt.Format(timeFormat), body, pkg.PackageID)
	}
	return err
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

func insertRevision(ctx context.Context, tx *sql.Tx, revision caption.CaptionRevision) error {
	body, err := encode(revision)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO revisions(revision_id, package_id, submitted_at, body) VALUES(?, ?, ?, ?)`, revision.RevisionID, revision.PackageID, revision.SubmittedAt.Format(timeFormat), body)
	return err
}

func replaceFindings(ctx context.Context, tx *sql.Tx, commit workflow.Commit) error {
	revisions := make(map[string]struct{})
	for _, finding := range commit.Findings {
		revisions[finding.RevisionID] = struct{}{}
	}
	if commit.Summary != nil {
		revisions[commit.Summary.RevisionID] = struct{}{}
	}
	if len(revisions) == 0 && commit.Package.CurrentRevisionID != "" {
		revisions[commit.Package.CurrentRevisionID] = struct{}{}
	}
	for revisionID := range revisions {
		if _, err := tx.ExecContext(ctx, `DELETE FROM findings WHERE revision_id = ?`, revisionID); err != nil {
			return err
		}
	}
	for _, finding := range commit.Findings {
		body, err := encode(finding)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO findings(finding_id, revision_id, status, body) VALUES(?, ?, ?, ?)`, finding.FindingID, finding.RevisionID, finding.Status, body); err != nil {
			return err
		}
	}
	return nil
}

func upsertSummary(ctx context.Context, tx *sql.Tx, summary caption.CheckSummary) error {
	body, err := encode(summary)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO check_summaries(revision_id, body) VALUES(?, ?) ON CONFLICT(revision_id) DO UPDATE SET body = excluded.body`, summary.RevisionID, body)
	return err
}

func insertReview(ctx context.Context, tx *sql.Tx, review caption.ReviewDecision) error {
	body, err := encode(review)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO reviews(decision_id, package_id, revision_id, decided_at, body) VALUES(?, ?, ?, ?, ?)`, review.DecisionID, review.PackageID, review.RevisionID, review.DecidedAt.Format(timeFormat), body)
	return err
}

func upsertManifest(ctx context.Context, tx *sql.Tx, manifest caption.ReleaseManifest) error {
	body, err := encode(manifest)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO manifests(manifest_id, package_id, issued_at, body) VALUES(?, ?, ?, ?)`, manifest.ManifestID, manifest.PackageID, manifest.IssuedAt.Format(timeFormat), body)
	return err
}

func insertEvent(ctx context.Context, tx *sql.Tx, event audit.Event) error {
	body, err := encode(event)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events(event_id, package_id, sequence, digest, previous_digest, body) VALUES(?, ?, ?, ?, ?, ?)`, event.EventID, event.PackageID, event.Sequence, event.Digest, event.PreviousDigest, body)
	return err
}

func insertIdempotency(ctx context.Context, tx *sql.Tx, record workflow.IdempotencyRecord) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO idempotency(key, command, package_id, response) VALUES(?, ?, ?, ?)`, record.Key, record.Command, record.PackageID, []byte(record.Response))
	return err
}
