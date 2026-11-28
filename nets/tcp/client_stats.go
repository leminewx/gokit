package tcp

import "net"

// Stats 定义 TCP 客户端的累计运行统计
type Stats struct {
	ReadBytes  int64 // 累计读取字节数
	WriteBytes int64 // 累计写入字节数
	Reconnects int64 // 成功重连次数
	DialErrors int64 // 拨号失败次数
}

// GetStats 返回运行统计快照
func (own *Client) GetStats() Stats {
	return Stats{
		ReadBytes:  own.readBytes.Load(),
		WriteBytes: own.writeBytes.Load(),
		Reconnects: own.reconnects.Load(),
		DialErrors: own.dialErrors.Load(),
	}
}

// IsClosed 判断客户端是否已关闭
func (own *Client) IsClosed() bool {
	return own.closed.Load()
}

// IsConnected 判断当前是否存在可用连接
func (own *Client) IsConnected() bool {
	return !own.closed.Load() && own.getConn() != nil
}

// GetLocalAddr 返回当前连接的本地地址，无连接时返回 nil
func (own *Client) GetLocalAddr() net.Addr {
	if conn := own.getConn(); conn != nil {
		return conn.LocalAddr()
	}

	return nil
}

// GetRemoteAddr 返回当前连接的远端地址，无连接时返回 nil
func (own *Client) GetRemoteAddr() net.Addr {
	if conn := own.getConn(); conn != nil {
		return conn.RemoteAddr()
	}

	return nil
}
