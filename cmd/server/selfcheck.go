package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/workflow"
)

type selfcheckClient struct {
	baseURL string
	client  *http.Client
}

func runSelfcheck(ctx context.Context, address string) error {
	client := &selfcheckClient{baseURL: "http://" + address, client: &http.Client{}}
	if err := client.expectPage(ctx); err != nil {
		return err
	}
	pkg := caption.CaptionPackage{}
	if err := client.post(ctx, "/api/v1/packages", workflow.CreatePackageInput{ProgramTitle: "自检无障碍节目", LanguageTag: "zh-CN", FrameRate: "25", TimecodeMode: "non_drop", CreatedBy: "editor.selfcheck", IdempotencyKey: "selfcheck-create-001"}, &pkg); err != nil {
		return err
	}
	if pkg.Version != 1 {
		return fmt.Errorf("建档版本异常: %d", pkg.Version)
	}
	revision := caption.CaptionRevision{}
	importInput := workflow.ImportRevisionInput{SourceName: "selfcheck-v1.json", SubmittedBy: "editor.selfcheck", ExpectedVersion: 1, IdempotencyKey: "selfcheck-import-001", Cues: []caption.CueInput{{CueID: "cue-001", Sequence: 1, StartTimecode: "00:00:01:00", EndTimecode: "00:00:01:12", Speaker: "旁白", Lines: []string{"好"}}}}
	if err := client.post(ctx, "/api/v1/packages/"+pkg.PackageID+"/revisions", importInput, &revision); err != nil {
		return err
	}
	check := workflow.CheckResult{}
	if err := client.post(ctx, "/api/v1/packages/"+pkg.PackageID+"/checks", workflow.RunCheckInput{ActorID: "editor.selfcheck", ExpectedVersion: 2, IdempotencyKey: "selfcheck-check-001"}, &check); err != nil {
		return err
	}
	if len(check.Findings) != 1 || check.Findings[0].RuleCode != caption.RuleDuration {
		return fmt.Errorf("自检期望一个显示时长问题，实际 %d", len(check.Findings))
	}
	finding := caption.QualityFinding{}
	if err := client.post(ctx, "/api/v1/packages/"+pkg.PackageID+"/exceptions", workflow.ExceptionInput{FindingID: check.Findings[0].FindingID, Reason: "节目片头节奏固定，单字提示需保持原始时长。", ActorID: "editor.selfcheck", ExpectedVersion: 3, IdempotencyKey: "selfcheck-except-001"}, &finding); err != nil {
		return err
	}
	decision := caption.ReviewDecision{}
	reviewInput := workflow.ReviewInput{ReviewerID: "reviewer.selfcheck", Outcome: caption.ReviewApproved, AcceptedExceptionIDs: []string{finding.FindingID}, Comment: "独立确认例外依据与节目画面一致。", ExpectedVersion: 4, IdempotencyKey: "selfcheck-review-001"}
	if err := client.post(ctx, "/api/v1/packages/"+pkg.PackageID+"/reviews", reviewInput, &decision); err != nil {
		return err
	}
	frozen := caption.CaptionPackage{}
	if err := client.post(ctx, "/api/v1/packages/"+pkg.PackageID+"/freeze", workflow.FreezeInput{ActorID: "release.selfcheck", ExpectedVersion: 5, IdempotencyKey: "selfcheck-freeze-001"}, &frozen); err != nil {
		return err
	}
	if frozen.FrozenDigest == "" {
		return fmt.Errorf("冻结摘要为空")
	}
	manifest := caption.ReleaseManifest{}
	if err := client.post(ctx, "/api/v1/packages/"+pkg.PackageID+"/manifest", workflow.IssueInput{IssuedBy: "release.selfcheck", ExpectedVersion: 6, IdempotencyKey: "selfcheck-issue-001"}, &manifest); err != nil {
		return err
	}
	var verification struct {
		Valid bool   `json:"valid"`
		Code  string `json:"code"`
	}
	if err := client.post(ctx, "/api/v1/manifests/verify", manifest, &verification); err != nil {
		return err
	}
	if !verification.Valid || verification.Code != "valid" {
		return fmt.Errorf("清单验证失败: %+v", verification)
	}
	var history struct {
		ChainValid bool  `json:"chainValid"`
		History    []any `json:"history"`
	}
	if err := client.get(ctx, "/api/v1/packages/"+pkg.PackageID+"/history", &history); err != nil {
		return err
	}
	if !history.ChainValid || len(history.History) != 7 {
		return fmt.Errorf("审计链不完整: %d", len(history.History))
	}
	return nil
}

func (c *selfcheckClient) expectPage(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/workbench", nil)
	if err != nil {
		return err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("访问工作台: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("字幕发布准入工作台")) {
		return fmt.Errorf("工作台页面自检失败: %s", response.Status)
	}
	return nil
}

func (c *selfcheckClient) post(ctx context.Context, path string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	return c.do(request, output)
}

func (c *selfcheckClient) get(ctx context.Context, path string, output any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(request, output)
}

func (c *selfcheckClient) do(request *http.Request, output any) error {
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("HTTP 自检 %s: %w", request.URL.Path, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP 自检 %s 返回 %s: %s", request.URL.Path, response.Status, strings.TrimSpace(string(body)))
	}
	if output != nil && len(body) > 0 {
		if err := json.Unmarshal(body, output); err != nil {
			return fmt.Errorf("解析自检响应 %s: %w", request.URL.Path, err)
		}
	}
	return nil
}
