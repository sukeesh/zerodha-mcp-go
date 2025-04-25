package internal

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

const (
	timeLayout = "2006-01-02 15:04:05"
)

type ZerodhaMcpServer struct {
	kc *kiteconnect.Client
}

func NewZerodhaMcpServer(kc *kiteconnect.Client) *ZerodhaMcpServer {
	return &ZerodhaMcpServer{
		kc: kc,
	}
}

func (z *ZerodhaMcpServer) SetKc(kc *kiteconnect.Client) {
	z.kc = kc
}

func printStruct(s interface{}) string {
	val := reflect.ValueOf(s)
	typ := reflect.TypeOf(s)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	returnVal := "<start> "

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldName := typ.Field(i).Name
		returnVal += fmt.Sprintf("%s: %v, ", fieldName, field.Interface())
	}
	returnVal += " <end>"

	return returnVal
}

func getHoldingText(holding kiteconnect.Holding) string {
	holdingTemplate := "Holding: Tradingsymbol: %s, Exchange: %s, InstrumentToken %d, ISIN %s, Product %s, Price %.2f, UsedQuantity %d, Quantity %d, T1Quantity %d, RealisedQuantity %d, Average Price %.2f, Last Price %.2f, Close Price %.2f, PnL %.2f, DayChange %.2f, DayChangePercentage %.2f, Buy Value: %.2f, Current Total Value: %.2f, MTFHolding: %x"
	return fmt.Sprintf(holdingTemplate, holding.Tradingsymbol, holding.Exchange, holding.InstrumentToken, holding.ISIN, holding.Product, holding.Price, holding.UsedQuantity, holding.Quantity, holding.T1Quantity, holding.RealisedQuantity, holding.AveragePrice, holding.LastPrice, holding.ClosePrice, holding.PnL, holding.DayChange, holding.DayChangePercentage, holding.AveragePrice*float64(holding.Quantity), holding.LastPrice*float64(holding.Quantity), holding.MTF)
}

func (z *ZerodhaMcpServer) KiteHoldingsTool() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		holdings, err := z.kc.GetHoldings()
		if err != nil {
			return nil, err
		}
		holdingsText := ""
		for _, holding := range holdings {
			eachHolding := getHoldingText(holding)
			holdingsText += eachHolding + "\n"
		}

		return mcp.NewToolResultText(holdingsText), nil
	}
}

func (z *ZerodhaMcpServer) AuctionInstrumentsTool() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		auctionInstruments, err := z.kc.GetAuctionInstruments()
		if err != nil {
			return nil, err
		}
		auctionInstrumentsText := ""
		for _, auctionInstrument := range auctionInstruments {
			eachAuctionInstrument := printStruct(auctionInstrument)
			auctionInstrumentsText += eachAuctionInstrument + "\n"
		}

		return mcp.NewToolResultText(auctionInstrumentsText), nil
	}
}

func (z *ZerodhaMcpServer) Positions() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		positions, err := z.kc.GetPositions()
		if err != nil {
			return nil, err
		}
		dayPositions := "DAY POSITIONS --- "
		for _, eachPosition := range positions.Day {
			eachPositionText := printStruct(eachPosition)
			dayPositions += eachPositionText + "\n"
		}
		netPositions := "NET POSITIONS --- "
		for _, position := range positions.Net {
			eachPosition := printStruct(position)
			netPositions += eachPosition + "\n"
		}

		positionsText := dayPositions + " \n \n " + netPositions
		return mcp.NewToolResultText(positionsText), nil
	}
}

func (z *ZerodhaMcpServer) OrderMargins() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// TODO: Currently accepting only single object, figure out a way with mcp.WithObject to deal with slice of objects

		exchange := request.Params.Arguments["exchange"].(string)
		tradingSymbol := request.Params.Arguments["tradingSymbol"].(string)
		transactionType := request.Params.Arguments["transactionType"].(string)
		variety := request.Params.Arguments["variety"].(string)
		product := request.Params.Arguments["product"].(string)
		orderType := request.Params.Arguments["orderType"].(string)
		quantity := request.Params.Arguments["quantity"].(float64)
		price := request.Params.Arguments["price"].(float64)
		triggerPrice := request.Params.Arguments["triggerPrice"].(float64)

		orderMargins, err := z.kc.GetOrderMargins(kiteconnect.GetMarginParams{
			OrderParams: []kiteconnect.OrderMarginParam{
				{
					Exchange:        exchange,
					Tradingsymbol:   tradingSymbol,
					TransactionType: transactionType,
					Variety:         variety,
					Product:         product,
					OrderType:       orderType,
					Quantity:        quantity,
					Price:           price,
					TriggerPrice:    triggerPrice,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		orderMarginsText := ""
		for _, orderMargin := range orderMargins {
			eachOrderMargin := printStruct(orderMargin)
			orderMarginsText += eachOrderMargin + "\n"
		}
		return mcp.NewToolResultText(orderMarginsText), nil
	}
}

func (z *ZerodhaMcpServer) Quote() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instrument := request.Params.Arguments["instrument"].(string)
		quote, err := z.kc.GetQuote(instrument)
		if err != nil {
			return nil, err
		}
		quoteStr := printStruct(quote)
		return mcp.NewToolResultText(quoteStr), nil
	}
}

func (z *ZerodhaMcpServer) LTP() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instrumentInterface, ok := request.Params.Arguments["instrument"]
		if !ok || instrumentInterface == nil {
			return nil, fmt.Errorf("instrument parameter is required")
		}

		instrument, ok := instrumentInterface.(string)
		if !ok {
			return nil, fmt.Errorf("instrument must be a string")
		}

		ltp, err := z.kc.GetLTP(instrument)
		if err != nil {
			return nil, err
		}
		ltpStr := printStruct(ltp)
		return mcp.NewToolResultText(ltpStr), nil
	}
}

func (z *ZerodhaMcpServer) OHLC() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instrument := request.Params.Arguments["instrument"].(string)
		ohlc, err := z.kc.GetOHLC(instrument)
		if err != nil {
			return nil, err
		}
		ohlcStr := printStruct(ohlc)
		return mcp.NewToolResultText(ohlcStr), nil
	}
}

func (z *ZerodhaMcpServer) HistoricalData() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instrumentToken := request.Params.Arguments["instrumentToken"].(int)
		interval := request.Params.Arguments["interval"].(string)
		fromDateStr := request.Params.Arguments["fromDate"].(string)
		toDateStr := request.Params.Arguments["toDate"].(string)
		continuousStr := request.Params.Arguments["continuous"].(string)
		oiStr := request.Params.Arguments["oi"].(string)

		fromDate, err := time.Parse(timeLayout, fromDateStr)
		if err != nil {
			return nil, err
		}

		toDate, err := time.Parse(timeLayout, toDateStr)
		if err != nil {
			return nil, err
		}

		continuous, err := strconv.ParseBool(continuousStr)
		if err != nil {
			return nil, err
		}

		oi, err := strconv.ParseBool(oiStr)
		if err != nil {
			return nil, err
		}

		historicalData, err := z.kc.GetHistoricalData(instrumentToken, interval, fromDate, toDate, continuous, oi)
		if err != nil {
			return nil, err
		}

		historicalDataStr := ""
		for _, candle := range historicalData {
			eachCandle := printStruct(candle)
			historicalDataStr += eachCandle + "\n"
		}
		return mcp.NewToolResultText(historicalDataStr), nil
	}
}

func (z *ZerodhaMcpServer) Instruments() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instruments, err := z.kc.GetInstruments()
		if err != nil {
			return nil, err
		}
		instrumentsText := ""
		for _, instrument := range instruments {
			eachInstrument := printStruct(instrument)
			instrumentsText += eachInstrument + "\n"
		}
		return mcp.NewToolResultText(instrumentsText), nil
	}
}

func (z *ZerodhaMcpServer) InstrumentsByExchange() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		exchange := request.Params.Arguments["exchange"].(string)
		instruments, err := z.kc.GetInstrumentsByExchange(exchange)
		if err != nil {
			return nil, err
		}
		instrumentsText := ""
		for _, instrument := range instruments {
			eachInstrument := printStruct(instrument)
			instrumentsText += eachInstrument + "\n"
		}
		return mcp.NewToolResultText(instrumentsText), nil
	}
}

func (z *ZerodhaMcpServer) MFInstruments() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instruments, err := z.kc.GetMFInstruments()
		if err != nil {
			return nil, err
		}
		instrumentsText := ""
		for _, instrument := range instruments {
			eachInstrument := printStruct(instrument)
			instrumentsText += eachInstrument + "\n"
		}
		return mcp.NewToolResultText(instrumentsText), nil
	}
}

func (z *ZerodhaMcpServer) MFOrders() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mfOrders, err := z.kc.GetMFOrders()
		if err != nil {
			return nil, err
		}
		mfOrdersText := ""
		for _, order := range mfOrders {
			eachOrder := printStruct(order)
			mfOrdersText += eachOrder + "\n"
		}
		return mcp.NewToolResultText(mfOrdersText), nil
	}
}

func (z *ZerodhaMcpServer) MFOrderInfo() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orderId := request.Params.Arguments["orderId"].(string)
		mfOrderInfo, err := z.kc.GetMFOrderInfo(orderId)
		if err != nil {
			return nil, err
		}
		mfOrderInfoStr := printStruct(mfOrderInfo)
		return mcp.NewToolResultText(mfOrderInfoStr), nil
	}
}

func (z *ZerodhaMcpServer) MfSipInfo() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sipId := request.Params.Arguments["sipId"].(string)
		mfSipInfo, err := z.kc.GetMFSIPInfo(sipId)
		if err != nil {
			return nil, err
		}
		mfSipInfoStr := printStruct(mfSipInfo)
		return mcp.NewToolResultText(mfSipInfoStr), nil
	}
}

func (z *ZerodhaMcpServer) MFHoldings() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		holdings, err := z.kc.GetMFHoldings()
		if err != nil {
			return nil, err
		}
		holdingsText := ""
		for _, holding := range holdings {
			eachHolding := printStruct(holding)
			holdingsText += eachHolding + "\n"
		}
		return mcp.NewToolResultText(holdingsText), nil
	}
}

func (z *ZerodhaMcpServer) MFHoldingInfo() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		isin := request.Params.Arguments["isin"].(string)
		holdingInfo, err := z.kc.GetMFHoldingInfo(isin)
		if err != nil {
			return nil, err
		}
		holdingInfoStr := printStruct(holdingInfo)
		return mcp.NewToolResultText(holdingInfoStr), nil
	}
}

func (z *ZerodhaMcpServer) MFAllottedISINs() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		allottedISINs, err := z.kc.GetMFAllottedISINs()
		if err != nil {
			return nil, err
		}
		allottedISINsText := fmt.Sprintf("%s", allottedISINs)
		return mcp.NewToolResultText(allottedISINsText), nil
	}
}

func (z *ZerodhaMcpServer) UserProfile() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userProfile, err := z.kc.GetUserProfile()
		if err != nil {
			return nil, err
		}
		userProfileStr := printStruct(userProfile)
		return mcp.NewToolResultText(userProfileStr), nil
	}
}

func (z *ZerodhaMcpServer) FullUserProfile() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userProfile, err := z.kc.GetFullUserProfile()
		if err != nil {
			return nil, err
		}
		userProfileStr := printStruct(userProfile)
		return mcp.NewToolResultText(userProfileStr), nil
	}
}

func (z *ZerodhaMcpServer) UserMargins() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userMargins, err := z.kc.GetUserMargins()
		if err != nil {
			return nil, err
		}
		userMarginsText := printStruct(userMargins)
		return mcp.NewToolResultText(userMarginsText), nil
	}
}

func (z *ZerodhaMcpServer) UserSegmentMargins() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		segment := request.Params.Arguments["segment"].(string)
		userSegmentMargins, err := z.kc.GetUserSegmentMargins(segment)
		if err != nil {
			return nil, err
		}
		userSegmentMarginsText := printStruct(userSegmentMargins)
		return mcp.NewToolResultText(userSegmentMarginsText), nil
	}
}
