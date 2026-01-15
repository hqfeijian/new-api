-- 添加回调相关字段到 tasks 表

-- 添加回调URL字段
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS callback_url VARCHAR(500) DEFAULT NULL;

-- 添加回调状态字段 (PENDING, SUCCESS, FAILED)
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS callback_status VARCHAR(20) DEFAULT NULL;

-- 添加回调重试次数字段
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS callback_retry_count INT DEFAULT 0;

-- 添加回调时间戳字段
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS callback_time BIGINT DEFAULT 0;

-- 创建索引提升查询性能
CREATE INDEX IF NOT EXISTS idx_tasks_callback_status ON tasks(callback_status);
CREATE INDEX IF NOT EXISTS idx_tasks_callback_time ON tasks(callback_time);

-- 注释说明
COMMENT ON COLUMN tasks.callback_url IS '任务完成后的回调URL';
COMMENT ON COLUMN tasks.callback_status IS '回调状态: PENDING-待回调, SUCCESS-回调成功, FAILED-回调失败';
COMMENT ON COLUMN tasks.callback_retry_count IS '回调重试次数';
COMMENT ON COLUMN tasks.callback_time IS '最后一次回调尝试的时间戳';
