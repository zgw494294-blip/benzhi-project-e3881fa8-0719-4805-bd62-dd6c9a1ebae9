package web

import (
	"net/http"

	"caption-release-gate/internal/workflow"
)

func (h *Handler) HandleListPackages(w http.ResponseWriter, r *http.Request) {
	packages, err := h.workflow.ListPackages(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"packages": packages})
}

func (h *Handler) HandleCreatePackage(w http.ResponseWriter, r *http.Request) {
	var input workflow.CreatePackageInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	pkg, err := h.workflow.CreatePackage(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pkg)
}

func (h *Handler) HandleGetPackage(w http.ResponseWriter, r *http.Request) {
	id, err := packageID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	view, err := h.workflow.GetPackage(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *Handler) HandleImportRevision(w http.ResponseWriter, r *http.Request) {
	id, err := packageID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var input workflow.ImportRevisionInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.PackageID = id
	revision, err := h.workflow.ImportRevision(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, revision)
}

func (h *Handler) HandleRunChecks(w http.ResponseWriter, r *http.Request) {
	id, err := packageID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var input workflow.RunCheckInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.PackageID = id
	result, err := h.workflow.RunChecks(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) HandleRequestException(w http.ResponseWriter, r *http.Request) {
	id, err := packageID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var input workflow.ExceptionInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.PackageID = id
	finding, err := h.workflow.RequestException(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, finding)
}

func (h *Handler) HandleReplacement(w http.ResponseWriter, r *http.Request) {
	id, err := packageID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var input workflow.ReplacementInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.PackageID = id
	revision, err := h.workflow.SubmitReplacement(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, revision)
}
