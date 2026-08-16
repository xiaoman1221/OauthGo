package providers

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const stateTTL = 10 * time.Minute

// stateStore 存储待校验的 OAuth state，防止 CSRF
var stateStore = struct {
	sync.RWMutex
	m map[string]time.Time
}{m: make(map[string]time.Time)}

// GenerateState 生成随机 state 并保存
func GenerateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := hex.EncodeToString(b)

	stateStore.Lock()
	stateStore.m[state] = time.Now()
	stateStore.Unlock()
	return state
}

// startStateJanitor 定期清理过期 state，避免未完成登录的 state 无限累积
func startStateJanitor() {
	go func() {
		ticker := time.NewTicker(stateTTL)
		defer ticker.Stop()
		for range ticker.C {
			stateStore.Lock()
			for s, created := range stateStore.m {
				if time.Since(created) > stateTTL {
					delete(stateStore.m, s)
				}
			}
			stateStore.Unlock()
		}
	}()
}

func init() {
	startStateJanitor()
}

// VerifyState 校验 state 是否有效（一次性）
func VerifyState(state string) bool {
	if state == "" {
		return false
	}
	stateStore.Lock()
	defer stateStore.Unlock()

	created, ok := stateStore.m[state]
	if !ok {
		return false
	}
	delete(stateStore.m, state)
	if time.Since(created) > stateTTL {
		return false
	}
	return true
}
