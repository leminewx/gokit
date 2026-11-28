// Package password 提供基于 bcrypt 的密码摘要与校验。
//
// bcrypt 自带随机盐并把参数编码进摘要，因此不需要额外保存盐值；代价因子
// （cost）决定单次计算量，每上调 1 计算量翻倍。默认使用 bcrypt.DefaultCost（10），
// 是「服务端可接受延迟」与「离线爆破成本」的常用折中。
//
// 注意：bcrypt 只取密码的前 72 字节参与运算，超长输入会直接返回错误而不是
// 静默截断——调用方应自行限制密码长度，不要依赖自动截断。
package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// 代价因子常量，语义与 golang.org/x/crypto/bcrypt 保持一致。
const (
	// DEFAULT_COST 为默认代价因子（10）
	DEFAULT_COST = bcrypt.DefaultCost
	// MIN_COST 为允许的最小代价因子（4），仅用于测试
	MIN_COST = bcrypt.MinCost
	// MAX_COST 为允许的最大代价因子（31），实际使用会非常慢
	MAX_COST = bcrypt.MaxCost
)

// Hash 用默认代价因子生成密码摘要。
func Hash(plainPassword []byte) ([]byte, error) {
	return HashWithCost(plainPassword, DEFAULT_COST)
}

// HashWithCost 用指定代价因子生成密码摘要。
//
// cost 必须是 [MIN_COST, MAX_COST] 之间的值，否则返回 ErrInvalidCost；
// 密码长度超过 72 字节时返回 bcrypt 的 ErrPasswordTooLong（不做截断）。
func HashWithCost(plainPassword []byte, cost int) ([]byte, error) {
	if cost < MIN_COST || cost > MAX_COST {
		return nil, wrapError(ErrInvalidCost, "%d", cost)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(plainPassword, cost)
	if err != nil {
		return nil, err
	}
	return hashedPassword, nil
}

// Verify 校验密码是否与摘要匹配。
//
// 匹配返回 nil；不匹配返回 ErrPasswordMismatch；摘要不是合法 bcrypt 格式
// （如误把明文当摘要、摘要被截断）返回 ErrInvalidHash。
func Verify(hashedPassword, plainPassword []byte) error {
	err := bcrypt.CompareHashAndPassword(hashedPassword, plainPassword)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return ErrPasswordMismatch
	default:
		return wrapError(ErrInvalidHash, "%v", err)
	}
}

// Match 是 Verify 的布尔封装，便于直接用作「校验通过与否」的判断。
func Match(hashedPassword, plainPassword []byte) bool {
	return Verify(hashedPassword, plainPassword) == nil
}

// IsRehashNeeded 报告摘要在当前代价因子下是否需要重新生成。
//
// 用于逐步提升 cost：登录校验通过后若 IsRehashNeeded 为 true，可用新 cost 重算
// 摘要并落盘。摘要格式非法时返回 true（视为必须重建）。
func IsRehashNeeded(hashedPassword []byte, cost int) bool {
	current, err := bcrypt.Cost(hashedPassword)
	if err != nil {
		return true
	}
	return current != cost
}
