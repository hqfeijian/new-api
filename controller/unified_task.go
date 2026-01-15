package controller

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// UnifiedTaskCreate 统一任务创建接口
func UnifiedTaskCreate(c *gin.Context) {
	var req dto.UnifiedTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.UnifiedTaskCreateResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	// 1. 获取模型映射
	mapping, err := service.GetModelMapping(req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.UnifiedTaskCreateResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// 2. 设置platform到context
	c.Set("platform", string(mapping.Platform))

	// 3. 解析input参数
	inputParams, err := service.ParseInputParameters(req.Model, req.Input)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.UnifiedTaskCreateResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid input parameters: " + err.Error(),
		})
		return
	}

	// 4. 将input参数转换为JSON字符串存储在request body
	inputJSON, _ := json.Marshal(inputParams)
	c.Set("unified_task_input", string(inputJSON))

	// 5. 暂存回调URL到context（后续在Insert时保存）
	if req.CallBackUrl != "" {
		c.Set("callback_url", req.CallBackUrl)
	}

	// 6. 构建RelayInfo
	info, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.UnifiedTaskCreateResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to generate relay info: " + err.Error(),
		})
		return
	}
	info.OriginModelName = req.Model
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{
		Action: mapping.Action,
	}

	// 7. 调用RelayTaskSubmit
	taskErr := relay.RelayTaskSubmit(c, info)
	if taskErr != nil {
		c.JSON(taskErr.StatusCode, dto.UnifiedTaskCreateResponse{
			Code:    taskErr.StatusCode,
			Message: taskErr.Message,
		})
		return
	}

	// 8. 从context获取创建的task信息
	taskId := c.GetString("created_task_id")

	c.JSON(http.StatusOK, dto.UnifiedTaskCreateResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data: &dto.UnifiedTaskResponseData{
			TaskId: taskId,
			Model:  req.Model,
			State:  "queued",
		},
	})
}

// UnifiedTaskFetch 统一任务查询接口（单个）
func UnifiedTaskFetch(c *gin.Context) {
	userId := c.GetInt("id")
	taskId := c.Param("taskId")

	if taskId == "" {
		c.JSON(http.StatusBadRequest, dto.UnifiedTaskFetchResponse{
			Code:    http.StatusBadRequest,
			Message: "taskId is required",
		})
		return
	}

	// 查询任务
	task, exist, err := model.GetByTaskId(userId, taskId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.UnifiedTaskFetchResponse{
			Code:    http.StatusInternalServerError,
			Message: "query task failed: " + err.Error(),
		})
		return
	}

	if !exist {
		c.JSON(http.StatusNotFound, dto.UnifiedTaskFetchResponse{
			Code:    http.StatusNotFound,
			Message: "task not found",
		})
		return
	}

	// 转换为统一响应格式
	data := convertTaskToUnifiedFormat(task)

	c.JSON(http.StatusOK, dto.UnifiedTaskFetchResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

// UnifiedTaskFetchBatch 统一任务批量查询接口
func UnifiedTaskFetchBatch(c *gin.Context) {
	userId := c.GetInt("id")
	taskIdsStr := c.Query("taskIds")

	if taskIdsStr == "" {
		c.JSON(http.StatusBadRequest, dto.UnifiedTaskBatchFetchResponse{
			Code:    http.StatusBadRequest,
			Message: "taskIds is required",
		})
		return
	}

	// 解析taskIds
	taskIdList := strings.Split(taskIdsStr, ",")
	if len(taskIdList) == 0 {
		c.JSON(http.StatusBadRequest, dto.UnifiedTaskBatchFetchResponse{
			Code:    http.StatusBadRequest,
			Message: "taskIds cannot be empty",
		})
		return
	}

	// 限制批量查询数量
	if len(taskIdList) > 50 {
		c.JSON(http.StatusBadRequest, dto.UnifiedTaskBatchFetchResponse{
			Code:    http.StatusBadRequest,
			Message: "taskIds count exceeds limit (max 50)",
		})
		return
	}

	// 转换为interface{}数组
	var taskIds []any
	for _, id := range taskIdList {
		trimmedId := strings.TrimSpace(id)
		if trimmedId != "" {
			taskIds = append(taskIds, trimmedId)
		}
	}

	if len(taskIds) == 0 {
		c.JSON(http.StatusBadRequest, dto.UnifiedTaskBatchFetchResponse{
			Code:    http.StatusBadRequest,
			Message: "no valid taskIds provided",
		})
		return
	}

	// 批量查询
	tasks, err := model.GetByTaskIds(userId, taskIds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.UnifiedTaskBatchFetchResponse{
			Code:    http.StatusInternalServerError,
			Message: "query tasks failed: " + err.Error(),
		})
		return
	}

	// 转换为统一响应格式
	var dataList []*dto.UnifiedTaskDetailData
	for _, task := range tasks {
		dataList = append(dataList, convertTaskToUnifiedFormat(task))
	}

	c.JSON(http.StatusOK, dto.UnifiedTaskBatchFetchResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    dataList,
	})
}

// convertTaskToUnifiedFormat 将Task模型转换为统一响应格式
func convertTaskToUnifiedFormat(task *model.Task) *dto.UnifiedTaskDetailData {
	data := &dto.UnifiedTaskDetailData{
		TaskId:    task.TaskID,
		Model:     task.Properties.OriginModelName,
		State:     mapTaskStatusToState(task.Status),
		Progress:  task.Progress,
		CreatedAt: task.CreatedAt,
	}

	// 如果没有OriginModelName，尝试从UpstreamModelName获取
	if data.Model == "" {
		data.Model = task.Properties.UpstreamModelName
	}

	// 根据状态设置结果或错误
	switch task.Status {
	case model.TaskStatusSuccess:
		data.CompletedAt = task.FinishTime
		// 解析result
		var result map[string]interface{}
		if err := common.Unmarshal(task.Data, &result); err == nil && len(result) > 0 {
			data.Result = result
		}
		// 如果Data为空但有FailReason（某些平台将URL存在FailReason中）
		if (data.Result == nil || len(data.Result) == 0) && task.FailReason != "" {
			if strings.HasPrefix(task.FailReason, "http") {
				data.Result = map[string]interface{}{
					"resultUrls": []string{task.FailReason},
				}
			}
		}

	case model.TaskStatusFailure:
		data.CompletedAt = task.FinishTime
		data.Error = &dto.TaskErrorInfo{
			Code:    "task_failed",
			Message: task.FailReason,
		}
	}

	return data
}

// mapTaskStatusToState 映射任务状态到统一状态
func mapTaskStatusToState(status model.TaskStatus) string {
	switch status {
	case model.TaskStatusNotStart, model.TaskStatusSubmitted, model.TaskStatusQueued:
		return "queued"
	case model.TaskStatusInProgress:
		return "processing"
	case model.TaskStatusSuccess:
		return "succeeded"
	case model.TaskStatusFailure:
		return "failed"
	default:
		return "unknown"
	}
}
