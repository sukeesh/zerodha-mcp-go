# Zerodha MCP Server

<p align="center">
  <strong>Protocol to communicate with your Zerodha data written in Golang</strong>
</p>

<p align="center">
  <img src="https://raw.githubusercontent.com/sukeesh/sukeesh.github.io/refs/heads/master/assets/img/Zerodha_MCP.png" alt="Zerodha MCP Logo" width="200" />
</p>

[![Go](https://github.com/sukeesh/zerodha-mcp/workflows/Go/badge.svg)](https://github.com/sukeesh/zerodha-mcp/actions)

## Overview
Zerodha MCP Server provides an implementation of the Claude MCP (Model Completion Protocol) interface for Zerodha trading data. This allows Claude AI to access your Zerodha trading account information directly.

## Prerequisites
- [Go](https://go.dev/doc/install) (version 1.21 or later)
- A [Zerodha Kite](https://kite.zerodha.com) trading account
- [Claude Desktop App](https://claude.ai/download)
- API credentials from the [Kite Connect developer portal](https://developers.kite.trade/apps)

## Installation

### Option 1: Using Go Install
```bash
go install github.com/sukeesh/zerodha-mcp@latest
```

### Option 2: Build from Source
```bash
git clone https://github.com/sukeesh/zerodha-mcp.git
cd zerodha-mcp
go install
```

The binary will be installed to your GOBIN directory, which should be in your PATH.

## Configuration

1. Get your `ZERODHA_API_KEY` and `ZERODHA_API_SECRET` from the [Kite Connect developer portal](https://developers.kite.trade/apps)

2. Set up a redirect URL in the Kite developer portal:
   ```
   http://127.0.0.1:5888/auth
   ```

3. Configure Claude Desktop:
   - Open Claude Desktop → Settings → Developer → Edit Config
   - Add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "zerodha": {
      "command": "<path-to-zerodha-mcp-binary>",
      "env": {
       "ZERODHA_API_KEY": "<api_key>",
       "ZERODHA_API_SECRET": "<api_secret>"
      }
    }
  }
}
```

4. Restart Claude Desktop. When prompted, authenticate with your Zerodha Kite credentials.

## Usage

After setup, you can interact with your Zerodha account data directly through Claude. For example:

- "Show me my current portfolio holdings"
- "What's my current margin availability?"
- "Give me the latest price for RELIANCE"
- "Show me my open positions with P&L"

## Available Features

Currently, this server exposes read-only (FETCH) endpoints for:

- Portfolio holdings
- Positions (day and net)
- Order margins
- Real-time quotes and LTP (Last Traded Price)
- OHLC data
- Instruments list
- Mutual fund information
- User profile and margins

## Limitations

- Only read operations are supported; trading is not yet available
- Authentication token expires daily and requires re-login

