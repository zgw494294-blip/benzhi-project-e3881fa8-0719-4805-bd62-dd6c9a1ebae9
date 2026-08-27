package web

import (
	"net/http"

	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/workflow"
)

func (h *Handler) HandleReview(w http.ResponseWriter, r *http.Request) {
	id, err := packageID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var input workflow.ReviewInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.PackageID = id
	decision, err := h.workflow.Review(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, decision)
}

func (h *Handler) HandleFreeze(w http.ResponseWriter, r *http.Request) {
	id, err := packageID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var input workflow.FreezeInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.PackageID = id
	pkg, err := h.workflow.Freeze(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

func (h *Handler) HandleIssueManifest(w http.ResponseWriter, r *http.Request) {
	id, err := packageID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var input workflow.IssueInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.PackageID = id
	manifest, err := h.workflow.IssueManifest(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, manifest)
}

func (h *Handler) HandleHistory(w http.ResponseWriter, r *http.Request) {
	id, err := packageID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	history, err := h.workflow.History(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": history, "chainValid": true})
}

func (h *Handler) HandleVerifyManifest(w http.ResponseWriter, r *http.Request) {
	var manifest caption.ReleaseManifest
	if err := decodeJSON(w, r, &manifest); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.workflow.VerifyManifest(r.Context(), manifest)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
