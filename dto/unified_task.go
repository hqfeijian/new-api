package dto

import "encoding/json"

// UnifiedTaskCreateRequest 统一任务创建请求
type UnifiedTaskCreateRequest struct {
	Model       string                 `json:"model" binding:"required"`
	CallBackUrl string                 `json:"callBackUrl,omitempty"`
	Input       map[string]interface{} `json:"input"` // 动态参数
}

// UnifiedTaskCreateResponse 统一任务创建响应
type UnifiedTaskCreateResponse struct {
	Code    int                      `json:"code"`
	Message string                   `json:"message"`
	Data    *UnifiedTaskResponseData `json:"data,omitempty"`
}

type UnifiedTaskResponseData struct {
	TaskId string `json:"taskId"`
	Model  string `json:"model"`
	State  string `json:"state"` // queued, processing, succeeded, failed
}

// UnifiedTaskFetchResponse 统一任务查询响应
type UnifiedTaskFetchResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    *UnifiedTaskDetailData `json:"data,omitempty"`
}

// UnifiedTaskBatchFetchResponse 统一任务批量查询响应
type UnifiedTaskBatchFetchResponse struct {
	Code    int                      `json:"code"`
	Message string                   `json:"message"`
	Data    []*UnifiedTaskDetailData `json:"data,omitempty"`
}

type UnifiedTaskDetailData struct {
	TaskId      string                 `json:"taskId"`
	Model       string                 `json:"model"`
	State       string                 `json:"state"`
	Progress    string                 `json:"progress,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       *TaskErrorInfo         `json:"error,omitempty"`
	CreatedAt   int64                  `json:"createdAt"`
	CompletedAt int64                  `json:"completedAt,omitempty"`
}

type TaskErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TaskCallbackPayload 回调数据格式
type TaskCallbackPayload struct {
	TaskId      string                 `json:"taskId"`
	Model       string                 `json:"model"`
	State       string                 `json:"state"`
	Progress    string                 `json:"progress,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	ResultJson  string                 `json:"resultJson,omitempty"`
	Error       *TaskCallbackError     `json:"error,omitempty"`
	Param       string                 `json:"param,omitempty"`
	FailCode    string                 `json:"failCode,omitempty"`
	FailMsg     string                 `json:"failMsg,omitempty"`
	CostTime    int64                  `json:"costTime,omitempty"`
	CreatedAt   int64                  `json:"createTime"`
	CompletedAt int64                  `json:"completeTime,omitempty"`
}

type TaskCallbackError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BuildCallbackPayload 构建回调数据
func BuildCallbackPayload(taskId, model, state, progress string, result map[string]interface{}, err *TaskErrorInfo, createdAt, completedAt int64) *TaskCallbackPayload {
	payload := &TaskCallbackPayload{
		TaskId:      taskId,
		Model:       model,
		State:       state,
		Progress:    progress,
		Result:      result,
		CreatedAt:   createdAt,
		CompletedAt: completedAt,
	}

	if err != nil {
		payload.Error = &TaskCallbackError{
			Code:    err.Code,
			Message: err.Message,
		}
		payload.FailCode = err.Code
		payload.FailMsg = err.Message
	}

	// 序列化result为JSON字符串
	if result != nil {
		if resultBytes, marshalErr := json.Marshal(result); marshalErr == nil {
			payload.ResultJson = string(resultBytes)
		}
	}

	return payload
}
