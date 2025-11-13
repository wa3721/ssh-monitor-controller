package command

import (
	"context"
	"time"
)

type operation func() error

type shouldRetryFunc func(error) bool

// Config 重试配置
type config struct {
	maxRetries   int
	initialDelay time.Duration
	maxDelay     time.Duration
	retryIf      shouldRetryFunc
}

// Retry 执行带指数退避的重试操作[1,4](@ref)
func Retry(ctx context.Context, operation operation, config config) error {
	var lastErr error

	for i := 0; i <= config.maxRetries; i++ {
		// 执行操作
		lastErr = operation()
		if lastErr == nil {
			return nil // 成功
		}

		if i == config.maxRetries || (config.retryIf != nil && !config.retryIf(lastErr)) {
			break
		}

		waitTime := config.initialDelay * time.Duration(1<<uint(i))
		if waitTime > config.maxDelay {
			waitTime = config.maxDelay
		}

		// 等待，但监听上下文取消
		select {
		case <-time.After(waitTime):
			// 继续下一次重试
		case <-ctx.Done():
			return ctx.Err() // 上下文被取消
		}
	}

	return lastErr
}

// 默认配置
var defaultConfig = config{
	maxRetries:   3,
	initialDelay: 1 * time.Second,
	maxDelay:     10 * time.Second,
}

// RetryWithDefault 使用默认配置的重试
func retryWithDefault(ctx context.Context, operation operation) error {
	return Retry(ctx, operation, defaultConfig)
}
