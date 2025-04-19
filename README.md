## Zerodha MCP Server

Zerodha MCP Server to communicate with your personal investments data.

## Setup

1. Go to [Claude desktop](https://claude.ai/download) -> Settings -> Developer and then click on `Edit Config` and paste the below in the `claude_desktop_config.json` file.

2. Get your `ZERODHA_API_KEY` and `ZERODHA_API_SECRET` from [Kite Connect developer portal](https://developers.kite.trade/apps). This is free from Zerodha.  

```json
{
  "mcpServers": {
    "zerodha": {
      "command": "<path-to-zerodha-mcp>",
      "env": {
       "ZERODHA_API_KEY": "<api_key>",
       "ZERODHA_API_SECRET": "<api_secret>"
      }
    }
  }
}
```

You can generate the `zerodha-mcp` binary by running `go install` in this directory.

3. Setup re-direct URL in the dev portal to the below
```bash
http://127.0.0.1:8080/auth
```

4. Restart Claude desktop app. It will ask you for authentication in the browser. Login with your Zerodha Kite credentials and start chatting on Claude.
