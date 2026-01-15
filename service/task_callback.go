package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

const (
	CallbackStatusPending = "PENDING"
	CallbackStatusSuccess = "SUCCESS"
	CallbackStatusFailed  = "FAILED"
)

// TriggerTaskCallback 触发任务回调
// 该函数应该在goroutine中异步调用
func TriggerTaskCallback(task *model.Task) {
	if task.CallBackUrl == "" {
		return
	}

	// 验证回调URL
	if !isValidCallbackURL(task.CallBackUrl) {
		logger.SysLog(fmt.Sprintf("Invalid callback URL for task %s: %s", task.TaskID, task.CallBackUrl))
		return
	}

	ctx := context.Background()

	// 构建回调payload
	payload := buildCallbackPayload(task)

	// 执行回调，最多重试3次
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := sendCallback(ctx, task.CallBackUrl, payload)
		if err == nil {
			// 回调成功，更新状态
			_ = model.DB.Model(&model.Task{}).
				Where("id = ?", task.ID).
				Updates(map[string]interface{}{
					"callback_status":      CallbackStatusSuccess,
					"callback_time":        time.Now().Unix(),
					"callback_retry_count": i + 1,
				}).Error
			logger.SysLog(fmt.Sprintf("Task callback success: %s, attempt: %d", task.TaskID, i+1))
			return
		}

		logger.SysLog(fmt.Sprintf("Task callback failed (attempt %d/%d): %s, error: %s",
			i+1, maxRetries, task.TaskID, err.Error()))

		// 更新重试次数
		_ = model.DB.Model(&model.Task{}).
			Where("id = ?", task.ID).
			Update("callback_retry_count", i+1).Error

		// 如果不是最后一次重试，等待后再试（指数退避）
		if i < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(1<<uint(i))) // 1s, 2s, 4s
		}
	}

	// 所有重试失败，标记为失败
	_ = model.DB.Model(&model.Task{}).
		Where("id = ?", task.ID).
		Updates(map[string]interface{}{
			"callback_status": CallbackStatusFailed,
			"callback_time":   time.Now().Unix(),
		}).Error
	logger.SysLog(fmt.Sprintf("Task callback failed after all retries: %s", task.TaskID))
}

func buildCallbackPayload(task *model.Task) *dto.TaskCallbackPayload {
	state := mapStatusToState(task.Status)
	model := task.Properties.OriginModelName
	if model == "" {
		model = task.Properties.UpstreamModelName
	}

	payload := &dto.TaskCallbackPayload{
		TaskId:    task.TaskID,
		Model:     model,
		State:     state,
		Progress:  task.Progress,
		CreatedAt: task.CreatedAt,
	}

	if task.Status == model.TaskStatusSuccess {
		payload.CompletedAt = task.FinishTime
		payload.CostTime = task.FinishTime - task.CreatedAt

		// 解析result
		var result map[string]interface{}
		if err := common.Unmarshal(task.Data, &result); err == nil && len(result) > 0 {
			payload.Result = result
			// 序列化为JSON字符串
			if resultBytes, marshalErr := json.Marshal(result); marshalErr == nil {
				payload.ResultJson = string(resultBytes)
			}
		}

		// 如果Data为空但有FailReason（某些平台将URL存在FailReason中）
		if (payload.Result == nil || len(payload.Result) == 0) && task.FailReason != "" {
			if strings.HasPrefix(task.FailReason, "http") {
				payload.Result = map[string]interface{}{
					"resultUrls": []string{task.FailReason},
				}
				if resultBytes, marshalErr := json.Marshal(payload.Result); marshalErr == nil {
					payload.ResultJson = string(resultBytes)
				}
			}
		}

	} else if task.Status == model.TaskStatusFailure {
		payload.CompletedAt = task.FinishTime
		payload.CostTime = task.FinishTime - task.CreatedAt
		payload.Error = &dto.TaskCallbackError{
			Code:    "task_failed",
			Message: task.FailReason,
		}
		payload.FailCode = "task_failed"
		payload.FailMsg = task.FailReason
	}

	// 保存请求参数（从Properties.Input）
	if task.Properties.Input != "" {
		payload.Param = task.Properties.Input
	}

	return payload
}

func sendCallback(ctx context.Context, callbackURL string, payload *dto.TaskCallbackPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", callbackURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "new-api-callback/1.0")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback returned status %d", resp.StatusCode)
	}

	return nil
}

func mapStatusToState(status model.TaskStatus) string {
	switch status {
	case model.TaskStatusSuccess:
		return "succeeded"
	case model.TaskStatusFailure:
		return "failed"
	case model.TaskStatusInProgress:
		return "processing"
	case model.TaskStatusQueued, model.TaskStatusSubmitted:
		return "queued"
	default:
		return string(status)
	}
}

// isValidCallbackURL 验证回调URL的安全性
func isValidCallbackURL(callbackURL string) bool {
	parsedURL, err := url.Parse(callbackURL)
	if err != nil {
		return false
	}

	// 必须是HTTP或HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	// 禁止内网地址
	host := parsedURL.Hostname()
	if host == "" {
		return false
	}

	// 禁止localhost和常见内网地址
	blacklist := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"::1",
	}

	for _, blocked := range blacklist {
		if strings.EqualFold(host, blocked) {
			return false
		}
	}

	// 禁止192.168.x.x和10.x.x.x内网段
	if strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") {
		return false
	}

	// 禁止172.16-31.x.x段
	if strings.HasPrefix(host, "172.") {
		parts := strings.Split(host, ".")
		if len(parts) >= 2 {
			// 检查第二位是否在16-31之间
			var second int
			if _, err := fmt.Sscanf(parts[1], "%d", &second); err == nil {
				if second >= 16 && second <= 31 {
					return false
				}
			}
		}
	}

	return true
}
