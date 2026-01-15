package service

import (
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/constant"
)

// ModelMappingConfig 模型映射配置
type ModelMappingConfig struct {
	Platform    constant.TaskPlatform
	Action      string
	ChannelType int
}

// modelMappingTable 模型名称到平台/Action的映射表
var modelMappingTable = map[string]ModelMappingConfig{
	// Sora系列
	"sora-2-pro-text-to-video": {ChannelType: constant.ChannelTypeSora, Action: constant.TaskActionGenerate},
	"sora-2-text-to-video":     {ChannelType: constant.ChannelTypeSora, Action: constant.TaskActionGenerate},

	// Kling系列
	"kling-v1":        {ChannelType: constant.ChannelTypeKling, Action: constant.TaskActionTextGenerate},
	"kling-v1-6":      {ChannelType: constant.ChannelTypeKling, Action: constant.TaskActionTextGenerate},
	"kling-v2-master": {ChannelType: constant.ChannelTypeKling, Action: constant.TaskActionTextGenerate},

	// Suno系列
	"suno_music":  {Platform: constant.TaskPlatformSuno, Action: constant.SunoActionMusic},
	"suno_lyrics": {Platform: constant.TaskPlatformSuno, Action: constant.SunoActionLyrics},

	// Ali通义千问系列
	"cogvideox-text-to-video":  {ChannelType: constant.ChannelTypeAli, Action: constant.TaskActionTextGenerate},
	"cogvideox-image-to-video": {ChannelType: constant.ChannelTypeAli, Action: constant.TaskActionGenerate},

	// Gemini系列
	"gemini-2.0-flash-exp":           {ChannelType: constant.ChannelTypeGemini, Action: constant.TaskActionTextGenerate},
	"gemini-exp-1206":                {ChannelType: constant.ChannelTypeGemini, Action: constant.TaskActionTextGenerate},
	"gemini-2.0-flash-thinking-exp": {ChannelType: constant.ChannelTypeGemini, Action: constant.TaskActionTextGenerate},

	// Hailuo (MiniMax)
	"hailuo-v1":      {ChannelType: constant.ChannelTypeMiniMax, Action: constant.TaskActionTextGenerate},
	"minimax-video": {ChannelType: constant.ChannelTypeMiniMax, Action: constant.TaskActionTextGenerate},

	// Vidu
	"vidu-1":         {ChannelType: constant.ChannelTypeVidu, Action: constant.TaskActionTextGenerate},
	"vidu-1-stable": {ChannelType: constant.ChannelTypeVidu, Action: constant.TaskActionTextGenerate},

	// Doubao (抖音)
	"doubao-video-pro":    {ChannelType: constant.ChannelTypeDoubaoVideo, Action: constant.TaskActionTextGenerate},
	"doubao-video-lite":   {ChannelType: constant.ChannelTypeDoubaoVideo, Action: constant.TaskActionTextGenerate},
	"doubao-video-turbo": {ChannelType: constant.ChannelTypeDoubaoVideo, Action: constant.TaskActionTextGenerate},

	// Jimeng (即梦)
	"jimeng-v2":    {ChannelType: constant.ChannelTypeJimeng, Action: constant.TaskActionTextGenerate},
	"jimeng-1.5-pro": {ChannelType: constant.ChannelTypeJimeng, Action: constant.TaskActionTextGenerate},

	// Vertex AI
	"vertex-imagen-3": {ChannelType: constant.ChannelTypeVertexAi, Action: constant.TaskActionTextGenerate},
}

// GetModelMapping 根据model获取映射配置
func GetModelMapping(model string) (*ModelMappingConfig, error) {
	if config, ok := modelMappingTable[model]; ok {
		// 如果Platform为空，则从ChannelType推导
		if config.Platform == "" && config.ChannelType > 0 {
			config.Platform = constant.TaskPlatform(strconv.Itoa(config.ChannelType))
		}
		return &config, nil
	}
	return nil, fmt.Errorf("unsupported model: %s", model)
}

// GetAllSupportedModels 获取所有支持的模型列表
func GetAllSupportedModels() []string {
	models := make([]string, 0, len(modelMappingTable))
	for model := range modelMappingTable {
		models = append(models, model)
	}
	return models
}

// ParseInputParameters 解析input参数
// 这个函数将动态的input map转换为通用的参数结构
// 不同的模型可能需要不同的参数，这里返回原始map供适配器使用
func ParseInputParameters(model string, input map[string]interface{}) (map[string]interface{}, error) {
	if input == nil {
		return make(map[string]interface{}), nil
	}

	// 参数验证：检查必需的参数
	_, err := GetModelMapping(model)
	if err != nil {
		return nil, err
	}

	// 返回原始输入参数，由具体的适配器处理
	// 适配器会根据自己的需求提取相应的参数
	return input, nil
}
