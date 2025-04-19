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

func startRouter() *http.Server {
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
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	return srv
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

func mcpMain(s *server.MCPServer, kc *kiteconnect.Client) {
	z = internal.NewZerodhaMcpServer(kc)

	kiteHoldingsTool := mcp.NewTool("fetch_holdings",
		mcp.WithDescription("Fetch current holdings of the user, These are stock market holdings, Index fund holdings only which are invested via NSE/BSE Directly."),
	)
	s.AddTool(kiteHoldingsTool, z.KiteHoldingsTool())

	auctionInstrumentsTool := mcp.NewTool("get_auction_instruments",
		mcp.WithDescription("Retrieves list of available instruments for a auction session"),
	)
	s.AddTool(auctionInstrumentsTool, z.AuctionInstrumentsTool())

	positionsTool := mcp.NewTool("get_positions",
		mcp.WithDescription("Retrieves list of DAY NET positions of the User on Zerodha"),
	)
	s.AddTool(positionsTool, z.Positions())

	orderMarginsTool := mcp.NewTool("get_order_margins",
		mcp.WithDescription("Retrieves list of order margins for a user"),
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
		mcp.WithDescription("gets map of quotes for given instruments in the format of `exchange:tradingsymbol`"),
	)
	s.AddTool(quoteTool, z.Quote())

	ltpTool := mcp.NewTool("get_ltp",
		mcp.WithDescription("gets LTP quote for given instrument in the format of `exchange:tradingsymbol`"),
		mcp.WithString("instrument",
			mcp.Required(),
			mcp.Description("format of `exchange:tradingsymbol`"),
		),
	)
	s.AddTool(ltpTool, z.LTP())

	ohlcTool := mcp.NewTool("get_ohlc",
		mcp.WithDescription("gets map of OHLC quotes for given instruments in the format of `exchange:tradingsymbol`"),
	)
	s.AddTool(ohlcTool, z.OHLC())

	// TODO: Historical data tool

	instrumentsTool := mcp.NewTool("get_instruments",
		mcp.WithDescription("retrieves list of instruments."),
	)
	s.AddTool(instrumentsTool, z.Instruments())

	instrumentsByExchange := mcp.NewTool("get_instruments_by_exchange",
		mcp.WithDescription("retrieves list of instruments by exchange"),
		mcp.WithString("exchange",
			mcp.Required(),
			mcp.Description("The exchange value"),
			mcp.Enum("nse", "bse"),
		),
	)
	s.AddTool(instrumentsByExchange, z.InstrumentsByExchange())

	mfInstruments := mcp.NewTool("get_mf_instruments",
		mcp.WithDescription("retrieves list of mutual fund instruments."),
	)
	s.AddTool(mfInstruments, z.MFInstruments())

	mfOrders := mcp.NewTool("get_mf_orders",
		mcp.WithDescription("retrieves list of Mutual Fund orders."),
	)
	s.AddTool(mfOrders, z.MFOrders())

	mfOrderInfo := mcp.NewTool("get_mf_order_info",
		mcp.WithDescription("get individual mutual fund order info."),
		mcp.WithString("orderId",
			mcp.Required(),
			mcp.Description("The Order ID of the mutual fund"),
		))
	s.AddTool(mfOrderInfo, z.MFOrderInfo())

	mfSipInfo := mcp.NewTool("get_mf_sip_info",
		mcp.WithDescription("get individual mutual fund order info."),
		mcp.WithString("sipId",
			mcp.Required(),
			mcp.Description("The SIP ID of the mutual fund"),
		))
	s.AddTool(mfSipInfo, z.MfSipInfo())

	mfHoldings := mcp.NewTool("get_mf_holdings",
		mcp.WithDescription("retrieves list of Mutual fund holdings for a user"),
	)
	s.AddTool(mfHoldings, z.MFHoldings())

	mfHoldingsInfo := mcp.NewTool("get_mf_holdings_info",
		mcp.WithDescription("get individual mutual fund holdings info."),
		mcp.WithString("isin",
			mcp.Required(),
			mcp.Description("The ISIN of the mutual fund holding"),
		))
	s.AddTool(mfHoldingsInfo, z.MFHoldingInfo())

	mfAllottedIsins := mcp.NewTool("get_mf_allotted_isins",
		mcp.WithDescription("get Allotted mutual fund ISINs."))
	s.AddTool(mfAllottedIsins, z.MFAllottedISINs())

	userProfile := mcp.NewTool("get_user_profile",
		mcp.WithDescription("get basic user profile"),
	)
	s.AddTool(userProfile, z.UserProfile())

	fullUserProfile := mcp.NewTool("get_full_user_profile",
		mcp.WithDescription("get full user profile"))
	s.AddTool(fullUserProfile, z.FullUserProfile())

	userMargins := mcp.NewTool("get_user_margins",
		mcp.WithDescription("get all user margins"))
	s.AddTool(userMargins, z.UserMargins())

	userSegmentMargins := mcp.NewTool("get_user_segment_margins",
		mcp.WithDescription("gets segment wise user margins."),
		mcp.WithString("segment",
			mcp.Required(),
			mcp.Description("segment of the mutual fund holding"),
		),
	)
	s.AddTool(userSegmentMargins, z.UserSegmentMargins())

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

	// Start the router in a goroutine and get the HTTP server
	var httpServer *http.Server
	httpServer = startRouter()

	kc := kiteAuthenticate()
	fmt.Fprintln(os.Stderr, "Zerodha authentication successful, starting MCP Server...")

	// Start the MCP server in a goroutine
	go func() {
		mcpMain(s, kc)
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		}
		log.Println("HTTP server stopped")
	}

	time.Sleep(1 * time.Second)
	log.Println("Server exiting")
	os.Exit(0)
}
