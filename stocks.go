package main

import (
	"embed"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jszwec/csvutil"
)

var (
	//go:embed chart.js index.html
	staticFS embed.FS
)

// Row in CSV
type Row struct {
	Date time.Time

	Close float64
	Open  float64
	High  float64
	Low   float64

	Volume int
}

// Table of data
type Table struct {
	Date []string

	Price []float64
	Open  []float64
	High  []float64
	Low   []float64

	Volume []int
}

type Exit struct {
	ExitDate  time.Time
	ExitPrice float64
}

type Trade struct {
	EnterDate  time.Time
	EnterPrice float64

	Exits []Exit

	Symbol     string
	ShowTrades bool
}

// unmarshalTime unmarshal data in CSV to time
func unmarshalTime(data []byte, t *time.Time) error {
	var err error
	*t, err = time.Parse("2006-01-02", string(data))
	return err
}

// parseData parses data from r and returns a table with columns filled
func parseData(r io.Reader) (Table, error) {
	dec, err := csvutil.NewDecoder(csv.NewReader(r))
	if err != nil {
		return Table{}, err
	}
	dec.Register(unmarshalTime)

	var table Table
	for {
		var row Row
		err := dec.Decode(&row)

		if err == io.EOF {
			break
		}

		if err != nil {
			return Table{}, err
		}

		table.Date = append(table.Date, row.Date.Format("2006-01-02"))
		table.Price = append(table.Price, math.Floor(row.Close*100)/100)
		table.Open = append(table.Open, math.Floor(row.Open*100)/100)
		table.High = append(table.High, math.Floor(row.High*100)/100)
		table.Low = append(table.Low, math.Floor(row.Low*100)/100)
		table.Volume = append(table.Volume, row.Volume)
	}

	return table, nil
}

// buildURL builds URL for downloading CSV from Yahoo! finance
func buildURL(symbol string, start, end time.Time) string {
	u := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/download/%s", url.PathEscape(symbol))
	v := url.Values{
		"period1":  {fmt.Sprintf("%d", start.Unix())},
		"period2":  {fmt.Sprintf("%d", end.Unix())},
		"interval": {"1d"},
		"events":   {"history"},
	}

	return fmt.Sprintf("%s?%s", u, v.Encode())
}

const (
	Daily = iota
	Weekly
	Monthly
)

// stockData returns stock data from Yahoo! finance
func stockData(symbol string, trade Trade, timeFrame int) (Table, error) {
	lastExitTrade := trade.Exits[0].ExitDate
	for i := range trade.Exits {
		if trade.Exits[i].ExitDate.After(lastExitTrade) {
			lastExitTrade = trade.Exits[i].ExitDate
		}
	}

	u := buildURL(symbol, trade.EnterDate.Add(-24*120*time.Hour), lastExitTrade.Add(24*120*time.Hour))
	resp, err := http.Get(u)
	if err != nil {
		return Table{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Table{}, fmt.Errorf("%s", resp.Status)
	}
	defer resp.Body.Close()

	table, err := parseData(resp.Body)
	if err != nil {
		return Table{}, err
	}

	if timeFrame == Weekly {
		return stockDataInWeekly(table), nil
	} else if timeFrame == Monthly {
		return stockDataInMonthly(table), nil
	}

	return table, nil
}

const weekEndDay = time.Friday

func stockDataInWeekly(table Table) Table {
	var weeklyData Table

	// Temporary variables to hold weekly data
	var weekOpen, weekHigh, weekLow, weekClose float64
	var weekVolume int
	var weekStarted bool

	for i, dateString := range table.Date {
		date, err := time.Parse("2006-01-02", dateString)
		if err != nil {
			fmt.Println("Error parsing date:", err)
			continue
		}

		// Set the Open price for the week
		if !weekStarted {
			weekOpen = table.Open[i]
			weekHigh = table.High[i]
			weekLow = table.Low[i]
			weekStarted = true
		}

		// Aggregate High and Low prices
		weekHigh = math.Max(weekHigh, table.High[i])
		weekLow = math.Min(weekLow, table.Low[i])

		// Update Close price and Volume
		weekClose = table.Price[i]
		weekVolume += table.Volume[i]

		// Check if it's the end of the week or the last day in the data
		if date.Weekday() == weekEndDay || i == len(table.Date)-1 {
			// Append the weekly data
			weeklyData.Date = append(weeklyData.Date, dateString)
			weeklyData.Open = append(weeklyData.Open, weekOpen)
			weeklyData.High = append(weeklyData.High, weekHigh)
			weeklyData.Low = append(weeklyData.Low, weekLow)
			weeklyData.Price = append(weeklyData.Price, weekClose)
			weeklyData.Volume = append(weeklyData.Volume, weekVolume)

			// Reset temporary variables for the next week
			weekHigh = 0
			weekLow = math.MaxFloat64
			weekVolume = 0
			weekStarted = false
		}
	}

	return weeklyData
}

func stockDataInMonthly(table Table) Table {
	var monthlyData Table

	// Temporary variables to hold monthly data
	var monthOpen, monthHigh, monthLow, monthClose float64
	var monthVolume int
	var monthStarted bool
	var currentMonth time.Month

	for i, dateString := range table.Date {
		date, err := time.Parse("2006-01-02", dateString)
		if err != nil {
			fmt.Println("Error parsing date:", err)
			continue
		}

		// Start a new month
		if !monthStarted || date.Month() != currentMonth {
			if monthStarted {
				// Append the previous month's data
				monthlyData.Date = append(monthlyData.Date, table.Date[i-1])
				monthlyData.Open = append(monthlyData.Open, monthOpen)
				monthlyData.High = append(monthlyData.High, monthHigh)
				monthlyData.Low = append(monthlyData.Low, monthLow)
				monthlyData.Price = append(monthlyData.Price, monthClose)
				monthlyData.Volume = append(monthlyData.Volume, monthVolume)
			}

			// Reset for the new month
			currentMonth = date.Month()
			monthOpen = table.Open[i]
			monthHigh = table.High[i]
			monthLow = table.Low[i]
			monthClose = table.Price[i]
			monthVolume = table.Volume[i]
			monthStarted = true
		} else {
			// Aggregate data for the month
			monthHigh = math.Max(monthHigh, table.High[i])
			monthLow = math.Min(monthLow, table.Low[i])
			monthClose = table.Price[i] // Last day's close price for the month
			monthVolume += table.Volume[i]
		}
	}

	if monthStarted {
		// Append the last month's data
		lastIndex := len(table.Date) - 1
		monthlyData.Date = append(monthlyData.Date, table.Date[lastIndex])
		monthlyData.Open = append(monthlyData.Open, monthOpen)
		monthlyData.High = append(monthlyData.High, monthHigh)
		monthlyData.Low = append(monthlyData.Low, monthLow)
		monthlyData.Price = append(monthlyData.Price, monthClose)
		monthlyData.Volume = append(monthlyData.Volume, monthVolume)
	}

	return monthlyData
}

// dataHandler returns JSON data for symbol
func dataHandler(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "empty symbol", http.StatusBadRequest)
		return
	}

	showTrades := false
	showTradesQuery := r.URL.Query().Get("showTrades")
	if showTradesQuery != "" {
		showTrades, _ = strconv.ParseBool(showTradesQuery)

	}

	tradeEnterDateQuery := r.URL.Query().Get("tradeEnterDate")
	if tradeEnterDateQuery == "" {
		log.Printf("data: %q", r.URL.Query().Get("tradeEnterDate"))
		http.Error(w, "empty tradeEnterDate", http.StatusBadRequest)
		return
	}

	tradeEnterDate, err := time.Parse("2006-01-02", tradeEnterDateQuery)
	if err != nil {
		log.Printf("data: %q", tradeEnterDateQuery)
		http.Error(w, "invalid tradeEnterDate", http.StatusBadRequest)
		return
	}

	tradeExitDateQuery := r.URL.Query().Get("tradeExitDate")
	if tradeExitDateQuery == "" {
		log.Printf("data: %q", r.URL.Query().Get("tradeExitDate"))
		http.Error(w, "empty tradeExitDate", http.StatusBadRequest)
		return
	}

	tradeExitDate, err := time.Parse("2006-01-02", tradeExitDateQuery)
	if err != nil {
		log.Printf("data: %q", tradeExitDateQuery)
		http.Error(w, "invalid tradeExitDate", http.StatusBadRequest)
		return
	}

	tradeEnterQuery := r.URL.Query().Get("buyPrice")
	if tradeEnterQuery == "" {
		log.Printf("data: %q", r.URL.Query().Get("tradeEnterPrice"))
		http.Error(w, "empty tradeEnterPrice", http.StatusBadRequest)
		return
	}

	tradeEnterPrice, err := strconv.ParseFloat(tradeEnterQuery, 64)
	if err != nil {
		log.Printf("data: %q", tradeEnterQuery)
		http.Error(w, "invalid tradeEnterDate", http.StatusBadRequest)
		return
	}

	trade := Trade{
		EnterDate:  tradeEnterDate,
		EnterPrice: tradeEnterPrice,
		Exits:      []Exit{},
		Symbol:     symbol,
		ShowTrades: showTrades,
	}

	exitPriceQuery := r.URL.Query().Get("exitPrice")
	if exitPriceQuery == "" {
		log.Printf("data: %q", r.URL.Query().Get("tradeExitPrice"))
		http.Error(w, "empty tradeExitPrice", http.StatusBadRequest)
		return
	}
	tradeExitPrice, err := strconv.ParseFloat(exitPriceQuery, 64)
	if err != nil {
		log.Printf("data: %q", exitPriceQuery)
		http.Error(w, "invalid tradeExitPrice", http.StatusBadRequest)
		return
	}
	trade.Exits = append(trade.Exits, Exit{
		ExitDate:  tradeExitDate,
		ExitPrice: tradeExitPrice,
	})

	tradeExitDateQuery2 := r.URL.Query().Get("tradeExitDate2")
	if tradeExitDateQuery2 != "" {
		tradeExitDate2, err := time.Parse("2006-01-02", tradeExitDateQuery2)
		if err != nil {
			http.Error(w, "invalid tradeExitDate2", http.StatusBadRequest)
			return
		}
		exitPriceQuery2 := r.URL.Query().Get("exitPrice2")
		if exitPriceQuery2 != "" {
			tradeExitPrice2, err := strconv.ParseFloat(exitPriceQuery2, 64)
			if err != nil {
				log.Printf("data: %q", exitPriceQuery2)
				http.Error(w, "invalid exitPrice2", http.StatusBadRequest)
				return
			}
			trade.Exits = append(trade.Exits, Exit{
				ExitDate:  tradeExitDate2,
				ExitPrice: tradeExitPrice2,
			})
		}

	}

	timeFrameMap := map[string]int{
		"daily":   Daily,
		"weekly":  Weekly,
		"monthly": Monthly,
		"":        Daily,
	}

	timeFrameQuery := r.URL.Query().Get("timeFrame")

	log.Printf("data: %q %q %q %f %f", symbol, tradeEnterDate, tradeExitDate, tradeEnterPrice, tradeExitPrice)

	table, err := stockData(symbol, trade, timeFrameMap[timeFrameQuery])
	if err != nil {
		log.Printf("get %q: %s", symbol, err)
		http.Error(w, "can't fetch data", http.StatusInternalServerError)
		return
	}

	if err := tableJSON(symbol, table, w, trade); err != nil {
		log.Printf("table: %s", err)
	}
}

// tableJSON writes table data as JSON into w
func tableJSON(symbol string, table Table, w io.Writer, trade Trade) error {
	colors := []string{"#00ff00", "#ff0000"}
	seriesData := make([]map[string]interface{}, len(table.Date))

	// Series data for volume
	volumeSeriesData := make([]map[string]interface{}, len(table.Date))

	for i := range table.Date {
		seriesData[i] = map[string]interface{}{
			"x": table.Date[i],
			"y": []float64{table.Open[i], table.High[i], table.Low[i], table.Price[i]},
		}

		volumeSeriesData[i] = map[string]interface{}{
			"x": table.Date[i],
			"y": table.Volume[i],
		}
	}

	var pointAnnotations []map[string]interface{}
	var xaxisAnnotations []map[string]interface{}

	if trade.ShowTrades {
		// Annotation for trade entry
		pointAnnotations = append(pointAnnotations, map[string]interface{}{
			"x":      trade.EnterDate.Format("2006-01-02"),
			"y":      trade.EnterPrice,
			"marker": map[string]interface{}{"size": 5},
			"label": map[string]interface{}{
				//"text":  "Enter",
				"style": map[string]interface{}{"background": "#fff"},
			},
		})

		// Process trade exits
		for i, exit := range trade.Exits {
			// Annotation for each trade exit
			pointAnnotations = append(pointAnnotations, map[string]interface{}{
				"x":      exit.ExitDate.Format("2006-01-02"),
				"y":      exit.ExitPrice,
				"marker": map[string]interface{}{"size": 5},
				"label": map[string]interface{}{
					"text":  "Exit",
					"style": map[string]interface{}{"background": "#fff"},
				},
			})

			// Creating x-axis range annotations
			var rangeStart time.Time
			if i == 0 {
				rangeStart = trade.EnterDate
			} else {
				rangeStart = trade.Exits[i-1].ExitDate
			}
			rangeEnd := exit.ExitDate

			xaxisAnnotations = append(xaxisAnnotations, map[string]interface{}{
				"x":         rangeStart.Format("2006-01-02"),
				"x2":        rangeEnd.Format("2006-01-02"),
				"fillColor": colors[i%2],
				"label": map[string]interface{}{
					"text": fmt.Sprintf("Buy %.2f", trade.EnterPrice),
				},
			})
		}
	}

	response := map[string]interface{}{
		"series": []interface{}{
			map[string]interface{}{
				"name": symbol,
				"data": seriesData,
			},
			map[string]interface{}{
				"name": "Volume",
				"data": volumeSeriesData,
				"type": "bar", // Specify the type of series for volume
			},
		},
		"annotations": map[string]interface{}{
			"points": pointAnnotations,
			"xaxis":  xaxisAnnotations,
		},
	}

	return json.NewEncoder(w).Encode(response)
}

func main() {
	http.Handle("/", http.FileServer(http.FS(staticFS)))
	http.HandleFunc("/data", dataHandler)
	http.HandleFunc("/upload", uploadFileHandler)
	http.HandleFunc("/trades", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "trades.html")
	})

	http.HandleFunc("/trade", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "trade.html")
	})

	http.HandleFunc("/visualiseTrader", tradeVisualisationHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func tradeVisualisationHandler(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "empty symbol", http.StatusBadRequest)
		return
	}

	showTrades := false
	showTradesQuery := r.URL.Query().Get("showTrades")
	if showTradesQuery != "" {
		showTrades, _ = strconv.ParseBool(showTradesQuery)

	}

	tradeEnterDateQuery := r.URL.Query().Get("tradeEnterDate")
	if tradeEnterDateQuery == "" {
		log.Printf("data: %q", r.URL.Query().Get("tradeEnterDate"))
		http.Error(w, "empty tradeEnterDate", http.StatusBadRequest)
		return
	}

	tradeEnterDate, err := time.Parse("2006-01-02", tradeEnterDateQuery)
	if err != nil {
		log.Printf("data: %q", tradeEnterDateQuery)
		http.Error(w, "invalid tradeEnterDate", http.StatusBadRequest)
		return
	}

	tradeExitDateQuery := r.URL.Query().Get("tradeExitDate")
	if tradeExitDateQuery == "" {
		log.Printf("data: %q", r.URL.Query().Get("tradeExitDate"))
		http.Error(w, "empty tradeExitDate", http.StatusBadRequest)
		return
	}

	tradeExitDate, err := time.Parse("2006-01-02", tradeExitDateQuery)
	if err != nil {
		log.Printf("data: %q", tradeExitDateQuery)
		http.Error(w, "invalid tradeExitDate", http.StatusBadRequest)
		return
	}

	tradeEnterQuery := r.URL.Query().Get("buyPrice")
	if tradeEnterQuery == "" {
		log.Printf("data: %q", r.URL.Query().Get("tradeEnterPrice"))
		http.Error(w, "empty tradeEnterPrice", http.StatusBadRequest)
		return
	}

	tradeEnterPrice, err := strconv.ParseFloat(tradeEnterQuery, 64)
	if err != nil {
		log.Printf("data: %q", tradeEnterQuery)
		http.Error(w, "invalid tradeEnterDate", http.StatusBadRequest)
		return
	}

	trade := Trade{
		EnterDate:  tradeEnterDate,
		EnterPrice: tradeEnterPrice,
		Exits:      []Exit{},
		Symbol:     symbol,
		ShowTrades: showTrades,
	}

	exitPriceQuery := r.URL.Query().Get("exitPrice")
	if exitPriceQuery == "" {
		log.Printf("data: %q", r.URL.Query().Get("tradeExitPrice"))
		http.Error(w, "empty tradeExitPrice", http.StatusBadRequest)
		return
	}
	tradeExitPrice, err := strconv.ParseFloat(exitPriceQuery, 64)
	if err != nil {
		log.Printf("data: %q", exitPriceQuery)
		http.Error(w, "invalid tradeExitPrice", http.StatusBadRequest)
		return
	}
	trade.Exits = append(trade.Exits, Exit{
		ExitDate:  tradeExitDate,
		ExitPrice: tradeExitPrice,
	})

	tradeExitDateQuery2 := r.URL.Query().Get("tradeExitDate2")
	if tradeExitDateQuery2 != "" {
		tradeExitDate2, err := time.Parse("2006-01-02", tradeExitDateQuery2)
		if err != nil {
			http.Error(w, "invalid tradeExitDate2", http.StatusBadRequest)
			return
		}
		exitPriceQuery2 := r.URL.Query().Get("exitPrice2")
		if exitPriceQuery2 != "" {
			tradeExitPrice2, err := strconv.ParseFloat(exitPriceQuery2, 64)
			if err != nil {
				log.Printf("data: %q", exitPriceQuery2)
				http.Error(w, "invalid exitPrice2", http.StatusBadRequest)
				return
			}
			trade.Exits = append(trade.Exits, Exit{
				ExitDate:  tradeExitDate2,
				ExitPrice: tradeExitPrice2,
			})
		}

	}

	timeFrameMap := map[string]int{
		"daily":   Daily,
		"weekly":  Weekly,
		"monthly": Monthly,
		"":        Daily,
	}

	timeFrameQuery := r.URL.Query().Get("timeFrame")

	log.Printf("data: %q %q %q %f %f", symbol, tradeEnterDate, tradeExitDate, tradeEnterPrice, tradeExitPrice)

	table, err := stockData(symbol, trade, timeFrameMap[timeFrameQuery])
	if err != nil {
		log.Printf("get %q: %s", symbol, err)
		http.Error(w, "can't fetch data", http.StatusInternalServerError)
		return
	}

	if err := tableJSON(symbol, table, w, trade); err != nil {
		log.Printf("table: %s", err)
	}

}

type TradeOrder struct {
	TradeID          string
	ExitID           string
	EntryDateTime    string // Keep as string for simplicity in HTTP handling
	ExitDateTime     string
	StockSymbol      string
	EntryType        string
	ExitType         string
	EntryQuantity    int
	ExitQuantity     int
	EntryPrice       float64
	ExitPrice        float64
	Commission       float64
	TotalCostForExit float64
	TraderID         string
	Market           string
	OrderStatus      string
	TradeDetailsLink string
}

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the multipart form
	r.ParseMultipartForm(10 << 20) // Limit file size to 10 MB

	// Retrieve the file from the form data
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Read the CSV file
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	if err != nil {
		http.Error(w, "Error reading the CSV file", http.StatusInternalServerError)
		return
	}

	tradeOrders := make([]TradeOrder, 0, len(records))
	// Process each record
	for _, record := range records {
		tradeOrder, err := parseCSVRecord(record)
		if err != nil {
			fmt.Println("Error parsing record:", err)
			continue
		}

		tradeOrders = append(tradeOrders, tradeOrder)
		// Process tradeOrder (e.g., store in database, perform calculations)
	}

	w.Header().Set("Content-Type", "application/json")
	// Encode the tradeOrders slice to JSON and send it as a response
	json.NewEncoder(w).Encode(tradeOrders)
}

func parseCSVRecord(record []string) (TradeOrder, error) {
	entryQuantity, _ := strconv.Atoi(record[7])
	exitQuantity, _ := strconv.Atoi(record[8])
	entryPrice, _ := strconv.ParseFloat(record[9], 64)
	exitPrice, _ := strconv.ParseFloat(record[10], 64)
	commission, _ := strconv.ParseFloat(record[11], 64)
	totalCostForExit, _ := strconv.ParseFloat(record[12], 64)

	// Constructing the link for trade details
	tradeDetailsLink := fmt.Sprintf(
		"http://localhost:8080/trade?symbol=%s&tradeEnterDate=%s&buyPrice=%f&tradeExitDate=%s&exitPrice=%f&tradeExitDate2=%s&exitPrice2=%f",
		record[4], // StockSymbol
		record[2], // EntryDateTime
		entryPrice,
		record[3], // ExitDateTime
		exitPrice,
		"",  // Placeholder for tradeExitDate2
		0.0, // Placeholder for exitPrice2
	)
	return TradeOrder{
		TradeID:          record[0],
		ExitID:           record[1],
		EntryDateTime:    record[2],
		ExitDateTime:     record[3],
		StockSymbol:      record[4],
		EntryType:        record[5],
		ExitType:         record[6],
		EntryQuantity:    entryQuantity,
		ExitQuantity:     exitQuantity,
		EntryPrice:       entryPrice,
		ExitPrice:        exitPrice,
		Commission:       commission,
		TotalCostForExit: totalCostForExit,
		TraderID:         record[13],
		Market:           record[14],
		OrderStatus:      record[15],
		TradeDetailsLink: tradeDetailsLink,
	}, nil
}
