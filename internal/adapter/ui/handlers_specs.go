package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nguyentin05/spectest/internal/infrastructure/suitefile"
)

type workspaceItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "openapi", "prd", "suite", "matrix", "other"
	Size int64  `json:"size"`
}

type workspaceResponse struct {
	CurrentDir string          `json:"current_dir"`
	Items      []workspaceItem `json:"items"`
}

type uploadResponse struct {
	Success  bool   `json:"success"`
	FileName string `json:"file_name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
}

// handleListWorkspace returns specs, PRD, and test suites in workspace.
func (s *Server) handleListWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workDir := s.workDir
	if workDir == "" {
		workDir = "."
	}
	absDir, _ := filepath.Abs(workDir)

	items := make([]workspaceItem, 0)
	_ = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := info.Name()
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		name := strings.ToLower(info.Name())

		itemType := ""
		if strings.Contains(name, "openapi") || strings.Contains(name, "swagger") || strings.Contains(name, "api-spec") {
			itemType = "openapi"
		} else if strings.Contains(name, "prd") || strings.Contains(name, "requirement") || (ext == ".md" && !strings.Contains(name, "matrix")) {
			itemType = "prd"
		} else if strings.HasSuffix(name, "spectest.yaml") || strings.HasSuffix(name, "spectest.yml") || strings.Contains(name, "suite") {
			itemType = "suite"
		} else if strings.Contains(name, "matrix") && ext == ".md" {
			itemType = "matrix"
		} else if ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".md" {
			itemType = "other"
		}

		if itemType != "" {
			rel, _ := filepath.Rel(absDir, path)
			items = append(items, workspaceItem{
				Name: rel,
				Path: path,
				Type: itemType,
				Size: info.Size(),
			})
		}
		return nil
	})

	writeJSON(w, http.StatusOK, workspaceResponse{
		CurrentDir: absDir,
		Items:      items,
	})
}

// handleUploadSpec handles uploading OpenAPI, PRD, or test suite files.
func (s *Server) handleUploadSpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Try Multipart Form
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
			writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "missing 'file' field in multipart form")
			return
		}
		defer func() {
			_ = file.Close()
		}()

		baseName := filepath.Base(header.Filename)
		destDir := filepath.Join(s.workDir, ".spectest_ui_uploads")
		_ = os.MkdirAll(destDir, 0o755)
		destPath := filepath.Join(destDir, baseName)

		out, err := os.Create(destPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save uploaded file: "+err.Error())
			return
		}
		defer func() {
			_ = out.Close()
		}()

		size, err := io.Copy(out, file)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to write file: "+err.Error())
			return
		}

		s.sessionMgr.AppendLog("Uploaded file: " + baseName)
		writeJSON(w, http.StatusOK, uploadResponse{
			Success:  true,
			FileName: baseName,
			Path:     destPath,
			Type:     detectFileType(baseName),
			Size:     size,
		})
		return
	}

	// 2. Try JSON raw content
	type rawUploadRequest struct {
		FileName string `json:"file_name"`
		Content  string `json:"content"`
	}

	var req rawUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	name := strings.TrimSpace(req.FileName)
	if name == "" {
		writeError(w, http.StatusBadRequest, "file_name is required")
		return
	}

	destDir := filepath.Join(s.workDir, ".spectest_ui_uploads")
	_ = os.MkdirAll(destDir, 0o755)
	destPath := filepath.Join(destDir, filepath.Base(name))

	if err := os.WriteFile(destPath, []byte(req.Content), 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write file: "+err.Error())
		return
	}

	s.sessionMgr.AppendLog("Saved spec file: " + name)
	writeJSON(w, http.StatusOK, uploadResponse{
		Success:  true,
		FileName: name,
		Path:     destPath,
		Type:     detectFileType(name),
		Size:     int64(len(req.Content)),
	})
}

// handleGetPlan loads and returns the parsed v1 structure of spectest.yaml.
func (s *Server) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	suitePath := r.URL.Query().Get("path")
	if suitePath == "" {
		candidates := []string{
			"spectest.yaml",
			"spectest.yml",
			filepath.Join(s.workDir, "spectest.yaml"),
			filepath.Join(s.workDir, "spectest.yml"),
			filepath.Join(s.workDir, "tests", "spectest.yaml"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				suitePath = c
				break
			}
		}
	}

	if suitePath == "" {
		writeError(w, http.StatusNotFound, "no spectest.yaml file found in workspace; please specify ?path= or upload one")
		return
	}

	suite, err := suitefile.LoadSuite(suitePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse test suite %q: %v", suitePath, err))
		return
	}

	totalSteps := 0
	for _, sc := range suite.Scenarios {
		totalSteps += len(sc.Steps)
	}
	for _, st := range suite.Setup {
		totalSteps += len(st.Steps)
	}

	lockedCount := 0
	skippedCount := 0
	for _, sc := range suite.Scenarios {
		if sc.Locked {
			lockedCount++
		}
		if sc.Skip != "" {
			skippedCount++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":      suitePath,
		"version":   suite.Version,
		"servers":   suite.Servers,
		"apis":      suite.APIs,
		"analysis":  suite.Analysis,
		"seed":      suite.Seed,
		"setup":     suite.Setup,
		"scenarios": suite.Scenarios,
		"stats": map[string]any{
			"total_scenarios":   len(suite.Scenarios),
			"total_setups":      len(suite.Setup),
			"total_steps":       totalSteps,
			"locked_scenarios":  lockedCount,
			"skipped_scenarios": skippedCount,
		},
	})
}

// handleGetHistory loads and returns the evolution history from spectest.log.yaml.
func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logPath := r.URL.Query().Get("path")
	if logPath == "" {
		candidates := []string{
			"spectest.log.yaml",
			"spectest.log.yml",
			filepath.Join(s.workDir, "spectest.log.yaml"),
			filepath.Join(s.workDir, "spectest.log.yml"),
			filepath.Join(s.workDir, "tests", "spectest.log.yaml"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				logPath = c
				break
			}
		}
	}

	if logPath == "" {
		writeError(w, http.StatusNotFound, "no spectest.log.yaml file found in workspace")
		return
	}

	logFile, err := suitefile.LoadLog(logPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse log file %q: %v", logPath, err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":    logPath,
		"base":    logFile.Base,
		"updates": logFile.Update,
	})
}

func detectFileType(name string) string {
	lower := strings.ToLower(name)
	ext := filepath.Ext(lower)
	if strings.Contains(lower, "openapi") || strings.Contains(lower, "swagger") {
		return "openapi"
	}
	if strings.Contains(lower, "prd") || (ext == ".md" && !strings.Contains(lower, "matrix")) {
		return "prd"
	}
	if strings.Contains(lower, "spectest.log") {
		return "history"
	}
	if strings.Contains(lower, "spectest") || ext == ".yaml" || ext == ".yml" {
		return "suite"
	}
	if strings.Contains(lower, "matrix") {
		return "matrix"
	}
	return "other"
}
