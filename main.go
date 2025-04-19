package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/toqueteos/webbrowser"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sukeesh/zerodha-mcp/internal"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

var (
	requestToken    = ""
	isAuthenticated = false
	z               *internal.ZerodhaMcpServer

	// Command line flags
	apiKey    string
	apiSecret string
)

func setEnvs() {
	eApiKey := os.Getenv("ZERODHA_API_KEY")
	eApiSecret := os.Getenv("ZERODHA_API_SECRET")

	// Validate required flags
	if eApiKey == "" || eApiSecret == "" {
		fmt.Println("Error: apikey and apisecret flags are required")
		fmt.Println("Usage example: ./zerodha-mcp -apikey=YOUR_API_KEY -apisecret=YOUR_API_SECRET")
		os.Exit(1)
	}

	apiKey = eApiKey
	apiSecret = eApiSecret
}

func renderHTMLResponse(c *gin.Context, content string, status int) {
	html := internal.RenderHTMLResponse(content)
	c.Data(status, "text/html; charset=utf-8", []byte(html))
}

func startRouter() (*http.Server, func()) {
	gin.DefaultWriter = os.Stderr
	gin.DefaultErrorWriter = os.Stderr

	// use a fresh Engine so we don't get the default stdout logger
	r := gin.New()
	r.Use(
		gin.LoggerWithWriter(os.Stderr),
		gin.RecoveryWithWriter(os.Stderr),
	)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/auth", func(c *gin.Context) {
		fmt.Fprintln(os.Stderr, "Request received on auth request")
		paramRequestToken := c.Query("request_token")
		if paramRequestToken == "" {
			// Render error template
			renderHTMLResponse(c, internal.ErrorContentTemplate, http.StatusBadRequest)
			return
		}

		requestToken = paramRequestToken
		if requestToken != "" {
			isAuthenticated = true
			// Render success template
			renderHTMLResponse(c, internal.SuccessContentTemplate, http.StatusOK)
		} else {
			// Render error template
			renderHTMLResponse(c, internal.ErrorContentTemplate, http.StatusBadRequest)
		}
	})

	srv := &http.Server{
		Addr:    ":5888",
		Handler: r,
	}

	// Create a shutdown function
	shutdownFn := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		log.Println("HTTP server stopped")
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	return srv, shutdownFn
}

func kiteAuthenticate() *kiteconnect.Client {
	isAuthenticated = false

	kc := kiteconnect.New(apiKey)
	webbrowser.Open(kc.GetLoginURL())

	curTime := time.Now()

	for {
		time.Sleep(5 * time.Second)
		if isAuthenticated {
			break
		} else {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Waiting for authentication from user. Please authenticate from %s", kc.GetLoginURL()))
		}

		if time.Since(curTime) > time.Minute*2 {
			fmt.Fprintln(os.Stderr, "2 mins and no auth yet for Zerodha, Exiting...")
			os.Exit(1)
		}
	}

	data, err := kc.GenerateSession(requestToken, apiSecret)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}

	kc.SetAccessToken(data.AccessToken)
	return kc
}

func mcpMain(ctx context.Context, s *server.MCPServer, kc *kiteconnect.Client) {
	z = internal.NewZerodhaMcpServer(kc)

	kiteHoldingsTool := mcp.NewTool("get_kite_holdings",
		mcp.WithDescription("Get current holdings in Zerodha Kite account. This includes stocks, ETFs, and other securities traded on NSE/BSE exchanges. Does not include mutual fund holdings."),
	)
	s.AddTool(kiteHoldingsTool, z.KiteHoldingsTool())

	auctionInstrumentsTool := mcp.NewTool("get_auction_instruments",
		mcp.WithDescription("Retrieves list of available instruments for a auction session"),
	)
	s.AddTool(auctionInstrumentsTool, z.AuctionInstrumentsTool())

	positionsTool := mcp.NewTool("get_positions",
		mcp.WithDescription("Get current day and net positions in your Zerodha account. Day positions show intraday trades, while net positions show delivery holdings and carried forward F&O positions. Includes quantity, average price, PnL and more details for each position."),
	)
	s.AddTool(positionsTool, z.Positions())

	orderMarginsTool := mcp.NewTool("get_order_margins",
		mcp.WithDescription("Get order margins for a specific instrument. This tool helps you check the margin requirements for placing orders on Zerodha. It provides the necessary information to ensure you have enough margin to execute trades."),
		mcp.WithString("exchange",
			mcp.Required(),
			mcp.Description("The exchange value"),
			mcp.Enum("nse", "bse"),
		),
		mcp.WithString("tradingSymbol",
			mcp.Required(),
			mcp.Description("The trading symbol"),
		),
		mcp.WithString("transactionType",
			mcp.Required(),
			mcp.Description("The transaction type"),
		),
		mcp.WithString("variety",
			mcp.Required(),
			mcp.Description("Variety"),
		),
		mcp.WithString("product",
			mcp.Required(),
			mcp.Description("Product"),
		),
		mcp.WithString("orderType",
			mcp.Required(),
			mcp.Description("Order Type"),
		),
		mcp.WithNumber("quantity",
			mcp.Required(),
			mcp.Description("Quantity"),
		),
		mcp.WithNumber("price",
			mcp.Required(),
			mcp.Description("Price"),
		),
		mcp.WithNumber("triggerPrice",
			mcp.Required(),
			mcp.Description("Trigger Price"),
		),
	)
	s.AddTool(orderMarginsTool, z.OrderMargins())

	quoteTool := mcp.NewTool("get_quote",
		mcp.WithDescription("Get quote for a specific instrument. This tool provides real-time market data for stocks, ETFs, and other securities traded on NSE/BSE exchanges."),
	)
	s.AddTool(quoteTool, z.Quote())

	ltpTool := mcp.NewTool("get_ltp",
		mcp.WithDescription("Get Last Traded Price (LTP) for a specific instrument. This tool provides the latest price at which the instrument was traded in the market."),
		mcp.WithString("instrument",
			mcp.Required(),
			mcp.Description("format of `exchange:tradingsymbol`"),
		),
	)
	s.AddTool(ltpTool, z.LTP())

	ohlcTool := mcp.NewTool("get_ohlc",
		mcp.WithDescription("Get Open, High, Low, Close (OHLC) quotes for a specific instrument. This tool provides the historical price data for the instrument over a specific time period."),
	)
	s.AddTool(ohlcTool, z.OHLC())

	// TODO: Complete Historical data tool. Need a way to consume huge amount of data.

	instrumentsTool := mcp.NewTool("get_instruments",
		mcp.WithDescription("Get list of all available instruments on Zerodha. This tool provides a comprehensive list of all the instruments that can be traded on Zerodha, including stocks, ETFs, futures, options, and more."),
	)
	s.AddTool(instrumentsTool, z.Instruments())

	instrumentsByExchange := mcp.NewTool("get_instruments_by_exchange",
		mcp.WithDescription("Get list of instruments by exchange. This tool allows you to filter and retrieve specific instruments based on the exchange they are traded on."),
		mcp.WithString("exchange",
			mcp.Required(),
			mcp.Description("The exchange value"),
			mcp.Enum("nse", "bse"),
		),
	)
	s.AddTool(instrumentsByExchange, z.InstrumentsByExchange())

	mfInstruments := mcp.NewTool("get_mf_instruments",
		mcp.WithDescription("Get list of all available mutual fund instruments on Zerodha. This tool provides a comprehensive list of all the mutual fund instruments that can be traded on Zerodha."),
	)
	s.AddTool(mfInstruments, z.MFInstruments())

	mfOrders := mcp.NewTool("get_mf_orders",
		mcp.WithDescription("Get list of all Mutual Fund orders. This tool provides a comprehensive list of all the mutual fund orders that can be traded on Zerodha."),
	)
	s.AddTool(mfOrders, z.MFOrders())

	mfOrderInfo := mcp.NewTool("get_mf_order_info",
		mcp.WithDescription("Get individual mutual fund order info. This tool provides detailed information about a specific mutual fund order, including the order ID, status, and other relevant details."),
		mcp.WithString("orderId",
			mcp.Required(),
			mcp.Description("The Order ID of the mutual fund"),
		))
	s.AddTool(mfOrderInfo, z.MFOrderInfo())

	mfSipInfo := mcp.NewTool("get_mf_sip_info",
		mcp.WithDescription("Get individual mutual fund SIP info. This tool provides detailed information about a specific mutual fund SIP, including the SIP ID, status, and other relevant details."),
		mcp.WithString("sipId",
			mcp.Required(),
			mcp.Description("The SIP ID of the mutual fund"),
		))
	s.AddTool(mfSipInfo, z.MfSipInfo())

	mfHoldings := mcp.NewTool("get_mf_holdings",
		mcp.WithDescription("Get list of Mutual fund holdings for a user. This tool provides a comprehensive list of all the mutual fund holdings that can be traded on Zerodha."),
	)
	s.AddTool(mfHoldings, z.MFHoldings())

	mfHoldingsInfo := mcp.NewTool("get_mf_holdings_info",
		mcp.WithDescription("Get individual mutual fund holdings info. This tool provides detailed information about a specific mutual fund holding, including the holding ID, status, and other relevant details."),
		mcp.WithString("isin",
			mcp.Required(),
			mcp.Description("The ISIN of the mutual fund holding"),
		))
	s.AddTool(mfHoldingsInfo, z.MFHoldingInfo())

	mfAllottedIsins := mcp.NewTool("get_mf_allotted_isins",
		mcp.WithDescription("Get Allotted mutual fund ISINs. This tool provides a comprehensive list of all the mutual fund ISINs that can be traded on Zerodha."))
	s.AddTool(mfAllottedIsins, z.MFAllottedISINs())

	userProfile := mcp.NewTool("get_user_profile",
		mcp.WithDescription("Get basic user profile. This tool provides basic information about the user, including the user ID, name, and other relevant details."),
	)
	s.AddTool(userProfile, z.UserProfile())

	// TODO: Figure out the right permissions for this
	//fullUserProfile := mcp.NewTool("get_full_user_profile",
	//	mcp.WithDescription("get full user profile"))
	//s.AddTool(fullUserProfile, z.FullUserProfile())

	userMargins := mcp.NewTool("get_user_margins",
		mcp.WithDescription("Get all user margins. This tool provides a comprehensive list of all the margins that can be traded on Zerodha."))
	s.AddTool(userMargins, z.UserMargins())

	userSegmentMargins := mcp.NewTool("get_user_segment_margins",
		mcp.WithDescription("Get segment wise user margins. This tool provides a comprehensive list of all the margins that can be traded on Zerodha."),
		mcp.WithString("segment",
			mcp.Required(),
			mcp.Description("segment of the mutual fund holding"),
		),
	)
	s.AddTool(userSegmentMargins, z.UserSegmentMargins())

	// Start the server and handle interruption via context
	go func() {
		<-ctx.Done()
		// This will only happen when ctx is cancelled - implement any cleanup needed here
		log.Println("MCP server received shutdown signal")
	}()

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func main() {
	setEnvs()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	s := server.NewMCPServer(
		"Zerodha MCP Server",
		"0.0.1",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	// Start the router and get the shutdown function
	_, httpShutdownFn := startRouter()

	kc := kiteAuthenticate()
	fmt.Fprintln(os.Stderr, "Zerodha authentication successful, starting MCP Server...")

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Start the MCP server in a goroutine
	mcpDone := make(chan struct{})
	go func() {
		defer close(mcpDone)
		mcpMain(ctx, s, kc)
	}()

	// Wait for quit signal
	<-quit
	log.Println("Shutting down server...")

	// Cancel the context to signal all operations to stop
	cancel()

	// Call the HTTP server shutdown function
	httpShutdownFn()

	// Wait for the MCP server to finish or timeout
	select {
	case <-mcpDone:
		log.Println("MCP server stopped")
	case <-time.After(5 * time.Second):
		log.Println("MCP server shutdown timed out")
	}

	log.Println("Server exiting")
	os.Exit(0)
}
