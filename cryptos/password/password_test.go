package password

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// _TEST_COST 选最小代价因子，避免单测被 bcrypt 计算拖慢。
const _TEST_COST = MIN_COST

// TestHashAndVerify 正确密码通过、错误密码返回 ErrPasswordMismatch。
func TestHashAndVerify(t *testing.T) {
	t.Parallel()

	plainPassword := []byte("Cictci@2022-test")
	hashedPassword, err := HashWithCost(plainPassword, _TEST_COST)
	if err != nil {
		t.Fatalf("HashWithCost 失败: %v", err)
	}

	if err = Verify(hashedPassword, plainPassword); err != nil {
		t.Errorf("Verify 正确密码返回错误: %v", err)
	}
	if !Match(hashedPassword, plainPassword) {
		t.Error("Match 正确密码应为 true")
	}

	if err = Verify(hashedPassword, []byte("gokit-definitely-wrong-password")); !errors.Is(err, ErrPasswordMismatch) {
		t.Errorf("Verify 错误密码 = %v, want ErrPasswordMismatch", err)
	}
	if Match(hashedPassword, []byte("gokit-definitely-wrong-password")) {
		t.Error("Match 错误密码应为 false")
	}
}

// TestHashUsesDefaultCost 默认摘要的代价因子为 DEFAULT_COST。
func TestHashUsesDefaultCost(t *testing.T) {
	t.Parallel()

	hashedPassword, err := Hash([]byte("secret"))
	if err != nil {
		t.Fatalf("Hash 失败: %v", err)
	}

	cost, err := bcrypt.Cost(hashedPassword)
	if err != nil {
		t.Fatalf("解析摘要代价失败: %v", err)
	}
	if cost != DEFAULT_COST {
		t.Errorf("cost = %d, want %d", cost, DEFAULT_COST)
	}
}

// TestHashWithCostRejectsInvalid 代价因子越界直接拒绝，避免不必要的高开销计算。
func TestHashWithCostRejectsInvalid(t *testing.T) {
	t.Parallel()

	for _, cost := range []int{MIN_COST - 1, MAX_COST + 1, 0, -1} {
		if _, err := HashWithCost([]byte("secret"), cost); !errors.Is(err, ErrInvalidCost) {
			t.Errorf("HashWithCost(cost=%d) 错误 = %v, want ErrInvalidCost", cost, err)
		}
	}
}

// TestHashRejectsTooLongPassword 超过 72 字节的密码必须报错而不是静默截断。
func TestHashRejectsTooLongPassword(t *testing.T) {
	t.Parallel()

	longPassword := make([]byte, 73)
	for index := range longPassword {
		longPassword[index] = 'a'
	}

	if _, err := HashWithCost(longPassword, _TEST_COST); !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Errorf("超长密码错误 = %v, want bcrypt.ErrPasswordTooLong", err)
	}
}

// TestVerifyInvalidHash 非 bcrypt 摘要（如明文）返回 ErrInvalidHash。
func TestVerifyInvalidHash(t *testing.T) {
	t.Parallel()

	if err := Verify([]byte("not-a-bcrypt-digest"), []byte("secret")); !errors.Is(err, ErrInvalidHash) {
		t.Errorf("非法摘要错误 = %v, want ErrInvalidHash", err)
	}
	if Match([]byte("not-a-bcrypt-digest"), []byte("secret")) {
		t.Error("非法摘要不应匹配成功")
	}
}

// TestIsRehashNeeded 代价因子不一致或摘要非法时需要重建。
func TestIsRehashNeeded(t *testing.T) {
	t.Parallel()

	hashedPassword, err := HashWithCost([]byte("secret"), _TEST_COST)
	if err != nil {
		t.Fatalf("HashWithCost 失败: %v", err)
	}

	if IsRehashNeeded(hashedPassword, _TEST_COST) {
		t.Errorf("同一代价因子不应需要重建")
	}
	if !IsRehashNeeded(hashedPassword, _TEST_COST+1) {
		t.Errorf("代价因子提升时应需要重建")
	}
	if !IsRehashNeeded([]byte("broken"), _TEST_COST) {
		t.Errorf("非法摘要应需要重建")
	}
}
