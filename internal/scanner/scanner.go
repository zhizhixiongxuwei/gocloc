// Package scanner 提供并发扫描调度能力。
// 该层负责目录遍历、任务分发、并发执行和结果聚合，不负责语法解析细节。
package scanner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"gocloc/internal/languages"
	"gocloc/internal/model"
)

// Service 是扫描服务对象。
type Service struct {
	registry *languages.Registry
	workers  int
}

// scanTask 表示一个待分析文件任务。
type scanTask struct {
	absolutePath string
	displayPath  string
	analyzer     languages.Analyzer
}

// workerResult 表示 worker 的执行产物。
type workerResult struct {
	fileMetrics *model.FileMetrics
	scanError   *model.ScanError
}

// NewService 创建扫描服务。
func NewService(registry *languages.Registry, workers int) *Service {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &Service{
		registry: registry,
		workers:  workers,
	}
}

// ScanPath 扫描目录或单文件。
// 扫描过程默认并发执行，单文件解析过程采用流式读取。
func (s *Service) ScanPath(targetPath string) (model.ScanResult, error) {
	var result model.ScanResult

	trimmedPath := strings.TrimSpace(targetPath)
	if trimmedPath == "" {
		return result, errors.New("scan path is empty")
	}

	absoluteTarget, err := filepath.Abs(trimmedPath)
	if err != nil {
		return result, fmt.Errorf("resolve absolute path: %w", err)
	}

	info, err := os.Stat(absoluteTarget)
	if err != nil {
		return result, fmt.Errorf("stat path: %w", err)
	}

	result.ScannedPath = absoluteTarget

	tasks := make(chan scanTask, s.workers*4)
	results := make(chan workerResult, s.workers*4)
	walkErrChan := make(chan error, 1)

	var workerGroup sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		workerGroup.Add(1)
		go func() {
			defer workerGroup.Done()
			s.runWorker(tasks, results)
		}()
	}

	go func() {
		defer close(tasks)
		if info.IsDir() {
			walkErrChan <- s.enqueueDirectoryTasks(absoluteTarget, tasks)
			return
		}
		walkErrChan <- s.enqueueSingleFileTask(absoluteTarget, tasks)
	}()

	go func() {
		workerGroup.Wait()
		close(results)
	}()

	result.Files = make([]model.FileMetrics, 0)
	result.Errors = make([]model.ScanError, 0)

	for item := range results {
		if item.fileMetrics != nil {
			result.Files = append(result.Files, *item.fileMetrics)
		}
		if item.scanError != nil {
			result.Errors = append(result.Errors, *item.scanError)
		}
	}

	if walkErr := <-walkErrChan; walkErr != nil {
		return result, walkErr
	}

	s.buildSummaries(&result)
	return result, nil
}

// enqueueDirectoryTasks 遍历目录并把可识别语言文件推入任务队列。
func (s *Service) enqueueDirectoryTasks(root string, tasks chan<- scanTask) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		analyzer, ok := s.registry.AnalyzerForFile(path)
		if !ok {
			return nil
		}

		relativePath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relativePath = path
		}

		tasks <- scanTask{
			absolutePath: path,
			displayPath:  filepath.ToSlash(relativePath),
			analyzer:     analyzer,
		}
		return nil
	})
}

// enqueueSingleFileTask 在用户给定单文件路径时创建任务。
func (s *Service) enqueueSingleFileTask(filePath string, tasks chan<- scanTask) error {
	analyzer, ok := s.registry.AnalyzerForFile(filePath)
	if !ok {
		return fmt.Errorf("unsupported file extension: %s", filepath.Ext(filePath))
	}

	tasks <- scanTask{
		absolutePath: filePath,
		displayPath:  filepath.Base(filePath),
		analyzer:     analyzer,
	}
	return nil
}

// runWorker 执行真实的文件读取和语言 FSM 分析。
func (s *Service) runWorker(tasks <-chan scanTask, results chan<- workerResult) {
	for task := range tasks {
		file, openErr := os.Open(task.absolutePath)
		if openErr != nil {
			results <- workerResult{
				scanError: &model.ScanError{
					Path:  task.displayPath,
					Error: openErr.Error(),
				},
			}
			continue
		}

		metrics, analyzeErr := task.analyzer.Analyze(file)
		closeErr := file.Close()

		if analyzeErr != nil {
			results <- workerResult{
				scanError: &model.ScanError{
					Path:  task.displayPath,
					Error: analyzeErr.Error(),
				},
			}
			continue
		}

		if closeErr != nil {
			results <- workerResult{
				scanError: &model.ScanError{
					Path:  task.displayPath,
					Error: closeErr.Error(),
				},
			}
			continue
		}

		results <- workerResult{
			fileMetrics: &model.FileMetrics{
				Path:     task.displayPath,
				Language: task.analyzer.Name(),
				Metrics:  metrics,
			},
		}
	}
}

// buildSummaries 计算语言级汇总和总计信息。
func (s *Service) buildSummaries(result *model.ScanResult) {
	sort.Slice(result.Files, func(i int, j int) bool {
		return result.Files[i].Path < result.Files[j].Path
	})

	sort.Slice(result.Errors, func(i int, j int) bool {
		return result.Errors[i].Path < result.Errors[j].Path
	})

	byLanguage := make(map[string]*model.LanguageMetrics)
	result.Total = model.TotalMetrics{}

	for _, item := range result.Files {
		result.Total.AddFileMetrics(item.Metrics)

		summary, ok := byLanguage[item.Language]
		if !ok {
			summary = &model.LanguageMetrics{
				Language:   item.Language,
				Extensions: s.registry.ExtensionsForLanguage(item.Language),
			}
			byLanguage[item.Language] = summary
		}

		summary.Files++
		summary.Metrics.Add(item.Metrics)
	}

	result.Languages = make([]model.LanguageMetrics, 0, len(byLanguage))
	for _, item := range byLanguage {
		result.Languages = append(result.Languages, *item)
	}

	sort.Slice(result.Languages, func(i int, j int) bool {
		return result.Languages[i].Language < result.Languages[j].Language
	})
}
