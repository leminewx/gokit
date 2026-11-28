package sshx

import "golang.org/x/crypto/ssh"

// Terminal PTY 伪终端参数
//   - 默认终端类型 xterm
//   - 默认终端宽度 120
//   - 默认终端高度 40
//   - 默认输入速度 14400
//   - 默认输出速度 14400
type Terminal struct {
	Term        string // 终端类型
	Width       int    // 终端宽度
	Height      int    // 终端高度
	TtyOpIspeed uint32 // 输入速度
	TtyOpOspeed uint32 // 输出速度
}

// Init 初始化伪终端
func (own Terminal) Init(sess *ssh.Session) error {
	term := own.normalize()

	return sess.RequestPty(term.Term, term.Width, term.Height, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: term.TtyOpIspeed,
		ssh.TTY_OP_OSPEED: term.TtyOpOspeed,
	})
}

// normalize 返回填充默认值后的终端参数副本
func (own Terminal) normalize() Terminal {
	if own.Term == "" {
		own.Term = "xterm"
	}
	if own.Width < 1 {
		own.Width = 120
	}
	if own.Height < 1 {
		own.Height = 40
	}
	if own.TtyOpIspeed < 1 {
		own.TtyOpIspeed = 14400
	}
	if own.TtyOpOspeed < 1 {
		own.TtyOpOspeed = 14400
	}

	return own
}
