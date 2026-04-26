package index

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rillanai/rillan/internal/config"
)

const graphifyPrefix = "graphify/"

type graphifyGraph struct {
	Nodes []map[string]any `json:"nodes"`
	Edges []map[string]any `json:"edges"`
}

// DiscoverGraphifyFiles reads graphify artifacts from disk and returns them as
// SourceFile entries. It is deliberately tolerant of every non-fatal failure
// mode (missing path, malformed graph.json, unreadable markdown, broken
// symlinks): each is logged via slog.Default and the corresponding artifact
// is skipped so indexing and serving never fail because graphify is
// misbehaving. The function returns nil error in normal operation; a non-nil
// error is reserved for programmer bugs in the graphify config itself.
func DiscoverGraphifyFiles(cfg config.KnowledgeGraphConfig) ([]SourceFile, error) {
	if !cfg.Enabled || strings.TrimSpace(cfg.Path) == "" {
		return nil, nil
	}

	root, err := filepath.Abs(cfg.Path)
	if err != nil {
		slog.Warn("graphify: resolve path failed, skipping graph ingestion",
			"path", cfg.Path, "error", err.Error())
		return nil, nil
	}

	if info, err := os.Stat(root); err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("graphify: stat path failed, skipping graph ingestion",
				"path", root, "error", err.Error())
		}
		return nil, nil
	} else if !info.IsDir() {
		slog.Warn("graphify: configured path is not a directory, skipping graph ingestion",
			"path", root)
		return nil, nil
	}

	files := make([]SourceFile, 0)

	graphJSONPath := filepath.Join(root, "graph.json")
	if graphData, err := os.ReadFile(graphJSONPath); err == nil {
		content, parseErr := summarizeGraphJSON(graphData, cfg)
		if parseErr != nil {
			slog.Warn("graphify: graph.json malformed, skipping summary",
				"path", graphJSONPath, "error", parseErr.Error())
		} else {
			files = append(files, SourceFile{
				AbsolutePath: graphJSONPath,
				RelativePath: graphifyPrefix + "graph.json",
				Content:      content,
				SizeBytes:    int64(len(content)),
			})
		}
	} else if !os.IsNotExist(err) {
		slog.Warn("graphify: read graph.json failed, skipping summary",
			"path", graphJSONPath, "error", err.Error())
	}

	walkErr := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, entryErr error) error {
		if entryErr != nil {
			slog.Warn("graphify: walk entry failed, skipping",
				"path", filePath, "error", entryErr.Error())
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(entry.Name()) != ".md" {
			return nil
		}

		relPath, err := filepath.Rel(root, filePath)
		if err != nil {
			slog.Warn("graphify: compute relative path failed, skipping",
				"path", filePath, "error", err.Error())
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		data, err := os.ReadFile(filePath)
		if err != nil {
			slog.Warn("graphify: read markdown file failed, skipping",
				"path", filePath, "error", err.Error())
			return nil
		}
		content := normalizeContent(string(data))
		if strings.TrimSpace(content) == "" {
			return nil
		}

		files = append(files, SourceFile{
			AbsolutePath: filePath,
			RelativePath: graphifyPrefix + relPath,
			Content:      content,
			SizeBytes:    int64(len(data)),
		})
		return nil
	})
	if walkErr != nil {
		slog.Warn("graphify: walk aborted, returning files discovered so far",
			"path", root, "error", walkErr.Error())
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].RelativePath < files[j].RelativePath
	})
	return files, nil
}

func summarizeGraphJSON(data []byte, cfg config.KnowledgeGraphConfig) (string, error) {
	var graph graphifyGraph
	if err := json.Unmarshal(data, &graph); err != nil {
		return "", fmt.Errorf("parse graph.json: %w", err)
	}

	limit := cfg.MaxNodes
	if limit <= 0 {
		limit = config.DefaultConfig().KnowledgeGraph.MaxNodes
	}
	if limit > len(graph.Nodes) {
		limit = len(graph.Nodes)
	}

	lines := []string{
		fmt.Sprintf("nodes: %d", len(graph.Nodes)),
		fmt.Sprintf("edges: %d", len(graph.Edges)),
	}
	for i := 0; i < limit; i++ {
		node := graph.Nodes[i]
		lines = append(lines, fmt.Sprintf("node[%d]: id=%v label=%v type=%v", i, node["id"], node["label"], node["type"]))
	}

	return strings.Join(lines, "\n"), nil
}
