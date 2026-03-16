package email

import (
	"fmt"
)

func BuildInviteEmailBody(workspaceName, token string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Workspace Invite</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f5;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="560" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.1);">
          <tr>
            <td style="background-color:#18181b;padding:32px 40px;">
              <p style="margin:0;color:#ffffff;font-size:20px;font-weight:600;">Your App</p>
            </td>
          </tr>
          <tr>
            <td style="padding:40px;">
              <h1 style="margin:0 0 8px;font-size:24px;font-weight:700;color:#18181b;">You've been invited</h1>
              <p style="margin:0 0 32px;font-size:15px;color:#71717a;">You've been invited to join the workspace <strong>%s</strong>. Use the token below to accept.</p>
              <table width="100%%" cellpadding="0" cellspacing="0">
                <tr>
                  <td align="center" style="background-color:#f4f4f5;border-radius:8px;padding:24px;">
                    <p style="margin:0;font-size:14px;font-weight:600;letter-spacing:2px;color:#18181b;font-family:monospace;word-break:break-all;">%s</p>
                  </td>
                </tr>
              </table>
              <p style="margin:24px 0 0;font-size:13px;color:#a1a1aa;text-align:center;">This invite expires in <strong>48 hours</strong>.</p>
            </td>
          </tr>
          <tr>
            <td style="padding:24px 40px;border-top:1px solid #f4f4f5;">
              <p style="margin:0;font-size:12px;color:#a1a1aa;text-align:center;">&copy; 2024 Your App. All rights reserved.</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, workspaceName, token)
}
