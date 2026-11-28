package smtpx

import "fmt"

// HTML_HEAD_TEMPLATE 邮件公共头部模板（包含移动端适配与主流邮件客户端的兼容声明）
const HTML_HEAD_TEMPLATE = `
	<!--[if !mso]><!-->
	<meta http-equiv="X-UA-Compatible" content="IE=edge">
	<!--<![endif]-->
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
	<style type="text/css">
		/* 添加邮件客户端兼容前缀 */
		#outlook a {
			padding: 0;
		}
		
		/* 强制Windows 10邮件应用使用桌面样式 */
		.ExternalClass,
		.ReadMsgBody {
			width: 100%;
		}
		
		/* 修复Yahoo邮件显示问题 */
		.yshortcuts a {
			border-bottom: none !important;
		}
		
		/* 修复iOS缩放问题 */
		@media only screen and (max-width: 600px) {
			table[class="container"] {
				width: 100% !important;
			}
		}
	</style>
	
	<!--[if !mso]><!-->
	<style type="text/css">
		/* 移动端样式优化 */
		@media only screen and (max-width:599px) {
		}
	</style>
	<!--<![endif]-->
`

// GenerateResetPasswordHTML 生成“重置密码成功”通知邮件的 HTML 正文
//   - appName：应用名称
//   - appEmail：应用帮助的官方邮箱
//   - appCopyrightDate：应用版权的发布年月信息
//   - registeredEmail：注册用户的邮箱
//   - registeredUsername：注册用户的名称
//   - registeredPassword：注册用户的新密码
func GenerateResetPasswordHTML(appName, appEmail, appCopyrightDate, registeredEmail, registeredUsername, registeredPassword string) []byte {
	tmpl := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>%s - 账号注册成功</title>
		<style>
			* {
				margin: 0;
				padding: 0;
				box-sizing: border-box;
				font-family: 'Helvetica Neue', Arial, sans-serif;
			}
			body {
				background-color: #f5f7fa;
				color: #333;
				line-height: 1.6;
			}
			.container {
				max-width: 700px;
				margin: 30px auto;
				background: #ffffff;
				border-radius: 12px;
				overflow: hidden;
				box-shadow: 0 5px 15px rgba(0, 0, 0, 0.08);
			}
			.header {
				background: linear-gradient(135deg, #1890ff 0%%, #096dd9 100%%);
				padding: 30px 20px;
				text-align: center;
				color: white;
			}
			.title {
				font-size: 24px;
				margin-top: 15px;
				font-weight: 600;
			}
			.content {
				padding: 20px 30px 0 30px;
			}
			.card {
				background: #f9fbfd;
				border-radius: 10px;
				padding: 25px;
				margin: 20px 0;
				border: 1px solid #e8edf3;
			}
			.info-box {
				margin-bottom: 25px;
			}
			.info-title {
				font-size: 18px;
				color: #1890ff;
				margin-bottom: 15px;
				font-weight: 600;
				display: flex;
				align-items: center;
			}
			.info-title i {
				margin-right: 10px;
				font-size: 20px;
			}
			.info-title p {
				margin-right: 5px;
			}
			.credentials {
				background: #f0f7ff;
				padding: 20px;
				border-radius: 8px;
				border-left: 4px solid #1890ff;
				margin: 15px 0;
			}
			.credential-item {
				display: flex;
				margin-bottom: 12px;
			}
			.credential-label {
				font-weight: 600;
				min-width: 100px;
				color: #555;
			}
			.password {
				font-size: 20px;
				letter-spacing: 2px;
				color: #e74c3c;
				font-weight: bold;
				background: #fffaf0;
				padding: 8px 15px;
				border-radius: 6px;
				display: inline-block;
				margin-top: 8px;
				border: 1px dashed #faad14;
			}
			.btn-container {
				text-align: center;
				margin: 30px 0;
			}
			.login-btn {
				display: inline-block;
				background: #1890ff;
				color: white !important;
				text-decoration: none;
				padding: 14px 35px;
				border-radius: 30px;
				font-size: 16px;
				font-weight: 600;
				transition: all 0.3s ease;
				box-shadow: 0 4px 10px rgba(24, 144, 255, 0.3);
			}
			.login-btn:hover {
				background: #096dd9;
				transform: translateY(-2px);
				box-shadow: 0 6px 15px rgba(24, 144, 255, 0.4);
			}
			.warning {
				background: #fffbe6;
				border: 1px solid #ffe58f;
				border-radius: 8px;
				padding: 18px;
				margin: 25px 0;
				color: #5c3b00;
			}
			.warning-title {
				display: flex;
				align-items: center;
				font-weight: 600;
				margin-bottom: 10px;
				color: #d48806;
			}
			.steps {
				padding-left: 10px;
			}
			.steps li {
				margin-bottom: 10px;
			}
			.footer {
				text-align: center;
				padding: 15px;
				background: #f5f7fa;
				color: #777;
				font-size: 14px;
			}
			.support {
				margin-top: 10px;
			}
			.support a {
				color: #1890ff;
				text-decoration: none;
			}
		</style>
		%s
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1 class="title">%s 重置密码成功</h1>
			</div>
			
			<div class="content">
				<div class="info-box">
					<p>尊敬的 <strong>%s</strong>：</p>
					<p>&emsp;&emsp;您好！</p>
					<p>&emsp;&emsp;您的账号已重置密码成功。以下是您的新密码，请妥善保管：</p>
				</div>
				
				<div class="card">
					<div class="info-title">
						<i>📋</i> 账号凭证
					</div>
					
					<div class="credentials">
						<div class="credential-item">
							<span class="credential-label">账号邮箱：</span>
							<span>%s</span>
						</div>
						<div class="credential-item">
							<span class="credential-label">初始密码：</span>
							<div>
								<div class="password">%s</div>
							</div>
						</div>
					</div>
				</div>
				
				<div class="warning">
					<div class="warning-title">
						<i>⚠️</i> 安全提示
					</div>
					<p>为保障您的账号安全，请务必：</p>
					<ul class="steps">
						<li>首次登录后立即修改密码</li>
						<li>设置8位以上包含字母、数字和特殊字符的强密码</li>
						<li>不要向任何人透露您的密码</li>
						<li>定期更换密码以提高安全性</li>
					</ul>
				</div>
			</div>
			
			<div class="footer">
				<p>© %s %s - 版权所有</p>
				<p class="support">如果需要帮助？请联系 <a href="mailto:%s">%s</a></p>
				<p style="margin-top: 15px; font-size: 12px; color: #999;">
					此为系统自动发送邮件，请勿直接回复
				</p>
			</div>
		</div>
	</body>
</html>`, appName, HTML_HEAD_TEMPLATE, appName, registeredUsername, registeredEmail, registeredPassword, appCopyrightDate, appName, appEmail, appEmail)
	return []byte(tmpl)
}

// GenerateRegisterUserHTML 生成“账号注册成功”通知邮件的 HTML 正文
//   - appName：应用名称
//   - appEmail：应用帮助的官方邮箱
//   - appCopyrightDate：应用版权的发布年月信息
//   - registeredEmail：注册用户的邮箱
//   - registeredUsername：注册用户的名称
//   - registeredPassword：注册用户的初始密码
func GenerateRegisterUserHTML(appName, appEmail, appCopyrightDate, registeredEmail, registeredUsername, registeredPassword string) []byte {
	tmpl := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>%s - 账号注册成功</title>
		<style>
			* {
				margin: 0;
				padding: 0;
				box-sizing: border-box;
				font-family: 'Helvetica Neue', Arial, sans-serif;
			}
			body {
				background-color: #f5f7fa;
				color: #333;
				line-height: 1.6;
			}
			.container {
				max-width: 700px;
				margin: 30px auto;
				background: #ffffff;
				border-radius: 12px;
				overflow: hidden;
				box-shadow: 0 5px 15px rgba(0, 0, 0, 0.08);
			}
			.header {
				background: linear-gradient(135deg, #1890ff 0%%, #096dd9 100%%);
				padding: 30px 20px;
				text-align: center;
				color: white;
			}
			.title {
				font-size: 24px;
				margin-top: 15px;
				font-weight: 600;
			}
			.content {
				padding: 20px 30px 0 30px;
			}
			.card {
				background: #f9fbfd;
				border-radius: 10px;
				padding: 25px;
				margin: 20px 0;
				border: 1px solid #e8edf3;
			}
			.info-box {
				margin-bottom: 25px;
			}
			.info-title {
				font-size: 18px;
				color: #1890ff;
				margin-bottom: 15px;
				font-weight: 600;
				display: flex;
				align-items: center;
			}
			.info-title i {
				margin-right: 10px;
				font-size: 20px;
			}
			.info-title p {
				margin-right: 5px;
			}
			.credentials {
				background: #f0f7ff;
				padding: 20px;
				border-radius: 8px;
				border-left: 4px solid #1890ff;
				margin: 15px 0;
			}
			.credential-item {
				display: flex;
				margin-bottom: 12px;
			}
			.credential-label {
				font-weight: 600;
				min-width: 100px;
				color: #555;
			}
			.password {
				font-size: 20px;
				letter-spacing: 2px;
				color: #e74c3c;
				font-weight: bold;
				background: #fffaf0;
				padding: 8px 15px;
				border-radius: 6px;
				display: inline-block;
				margin-top: 8px;
				border: 1px dashed #faad14;
			}
			.btn-container {
				text-align: center;
				margin: 30px 0;
			}
			.login-btn {
				display: inline-block;
				background: #1890ff;
				color: white !important;
				text-decoration: none;
				padding: 14px 35px;
				border-radius: 30px;
				font-size: 16px;
				font-weight: 600;
				transition: all 0.3s ease;
				box-shadow: 0 4px 10px rgba(24, 144, 255, 0.3);
			}
			.login-btn:hover {
				background: #096dd9;
				transform: translateY(-2px);
				box-shadow: 0 6px 15px rgba(24, 144, 255, 0.4);
			}
			.warning {
				background: #fffbe6;
				border: 1px solid #ffe58f;
				border-radius: 8px;
				padding: 18px;
				margin: 25px 0;
				color: #5c3b00;
			}
			.warning-title {
				display: flex;
				align-items: center;
				font-weight: 600;
				margin-bottom: 10px;
				color: #d48806;
			}
			.steps {
				padding-left: 10px;
			}
			.steps li {
				margin-bottom: 10px;
			}
			.footer {
				text-align: center;
				padding: 15px;
				background: #f5f7fa;
				color: #777;
				font-size: 14px;
			}
			.support {
				margin-top: 10px;
			}
			.support a {
				color: #1890ff;
				text-decoration: none;
			}
		</style>
		%s
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1 class="title">%s 账号注册成功</h1>
			</div>
			
			<div class="content">
				<div class="info-box">
					<p>尊敬的 <strong>%s</strong>：</p>
					<p>&emsp;&emsp;您好！</p>
					<p>&emsp;&emsp;感谢您注册 %s，您的账号已成功开通。以下是您的重要账号信息，请妥善保管：</p>
				</div>
				
				<div class="card">
					<div class="info-title">
						<i>📋</i> 账号凭证
					</div>
					
					<div class="credentials">
						<div class="credential-item">
							<span class="credential-label">账号邮箱：</span>
							<span>%s</span>
						</div>
						<div class="credential-item">
							<span class="credential-label">初始密码：</span>
							<div>
								<div class="password">%s</div>
							</div>
						</div>
					</div>
				</div>
				
				<div class="warning">
					<div class="warning-title">
						<i>⚠️</i> 安全提示
					</div>
					<p>为保障您的账号安全，请务必：</p>
					<ul class="steps">
						<li>首次登录后立即修改密码</li>
						<li>设置8位以上包含字母、数字和特殊字符的强密码</li>
						<li>不要向任何人透露您的密码</li>
						<li>定期更换密码以提高安全性</li>
					</ul>
				</div>
			</div>
			
			<div class="footer">
				<p>© %s %s - 版权所有</p>
				<p class="support">如果需要帮助？请联系 <a href="mailto:%s">%s</a></p>
				<p style="margin-top: 15px; font-size: 12px; color: #999;">
					此为系统自动发送邮件，请勿直接回复
				</p>
			</div>
		</div>
	</body>
</html>`, appName, HTML_HEAD_TEMPLATE, appName, registeredUsername, appName, registeredEmail, registeredPassword, appCopyrightDate, appName, appEmail, appEmail)
	return []byte(tmpl)
}
