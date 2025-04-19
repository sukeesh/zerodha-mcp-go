package internal

const (
	HTMLHeaderTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Zerodha MCP Authentication</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 0;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
        }
        
        .container {
            background-color: white;
            border-radius: 10px;
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
            padding: 30px;
            text-align: center;
            max-width: 500px;
            width: 100%;
        }
        
        .success {
            color: #28a745;
        }
        
        .error {
            color: #dc3545;
        }
        
        h1 {
            margin-bottom: 20px;
            font-weight: 600;
        }
        
        p {
            margin-bottom: 25px;
            color: #6c757d;
            line-height: 1.6;
        }
        
        .icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        
        .btn {
            display: inline-block;
            background-color: #007bff;
            color: white;
            text-decoration: none;
            padding: 10px 20px;
            border-radius: 5px;
            transition: background-color 0.3s;
        }
        
        .btn:hover {
            background-color: #0069d9;
        }
        
        .zerodha-logo {
            margin-bottom: 20px;
            max-width: 150px;
        }
    </style>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.15.4/css/all.min.css">
</head>
<body>`

	HTMLFooterTemplate = `</body>
</html>`

	SuccessContentTemplate = `    <div class="container">
        <img src="https://zerodha.com/static/images/logo.svg" alt="Zerodha Logo" class="zerodha-logo">
        <div class="icon success">
            <i class="fas fa-check-circle"></i>
        </div>
        <h1 class="success">Authentication Successful!</h1>
        <p>Your Zerodha account has been successfully authenticated with MCP Server.</p>
        <p>You can now close this window and continue using the Zerodha MCP tools.</p>
    </div>`

	ErrorContentTemplate = `    <div class="container">
        <img src="https://zerodha.com/static/images/logo.svg" alt="Zerodha Logo" class="zerodha-logo">
        <div class="icon error">
            <i class="fas fa-times-circle"></i>
        </div>
        <h1 class="error">Authentication Failed</h1>
        <p>There was a problem authenticating your Zerodha account with MCP Server.</p>
        <p>Please try again or contact support if the issue persists.</p>
        <a href="javascript:window.close();" class="btn">Close Window</a>
    </div>`
)

// RenderHTMLResponse combines HTML parts and returns the complete HTML string
func RenderHTMLResponse(content string) string {
	return HTMLHeaderTemplate + content + HTMLFooterTemplate
}
