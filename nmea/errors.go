package nmea

import "errors"

const _ERROR_PREFIX = "gokit.nmea: "

var (
	// ErrNotGGASentence 表示输入不是 GGA 语句
	ErrNotGGASentence = errors.New(_ERROR_PREFIX + "sentence is not GGA")
	// ErrGGACountMismatch 表示 GGA 字段数量不是 15
	ErrGGACountMismatch = errors.New(_ERROR_PREFIX + "invalid GGA field count")
	// ErrNotRMCSentence 表示输入不是 RMC 语句
	ErrNotRMCSentence = errors.New(_ERROR_PREFIX + "sentence is not RMC")
	// ErrRMCCountMismatch 表示 RMC 字段数量不是 13
	ErrRMCCountMismatch = errors.New(_ERROR_PREFIX + "invalid RMC field count")
	// ErrMissingChecksum 表示语句缺少 "*校验和" 片段
	ErrMissingChecksum = errors.New(_ERROR_PREFIX + "sentence is missing checksum")
)
