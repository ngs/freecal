// Google Calendar API（OAuth2）でイベントを取得し、
// 平日 9:00–17:00 の「連続 min 分以上の空き」を Markdown で出力します。
// 同日の複数スロットはカンマ区切り、日本語曜日を付与します。
// 例:
//
//	go mod init example.com/freecalapi
//	go get google.golang.org/api/calendar/v3 google.golang.org/api/option golang.org/x/oauth2 golang.org/x/oauth2/google
//
// 実行例:
//
//	go run ./cmd/freecalapi \
//	  -credentials ./credentials.json \
//	  -token ./token.json \
//	  -calendar primary \
//	  -start 2025-08-11 -end 2025-08-14 \
//	  -workstart 09:00 -workend 17:00 \
//	  -min 60 \
//	  -tz Asia/Tokyo
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type interval struct {
	start time.Time
	end   time.Time
}

func mustParseClock(s string) (h, m int) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		log.Fatalf("invalid time %q (want HH:MM): %v", s, err)
	}
	return t.Hour(), t.Minute()
}

const sundayJP = "日"

func formatJpWeekday(t time.Time) string {
	switch t.Weekday() {
	case time.Monday:
		return "月"
	case time.Tuesday:
		return "火"
	case time.Wednesday:
		return "水"
	case time.Thursday:
		return "木"
	case time.Friday:
		return "金"
	case time.Saturday:
		return "土"
	case time.Sunday:
		return sundayJP
	default:
		return sundayJP
	}
}

func overlaps(a, b interval) (interval, bool) {
	s := a.start
	if b.start.After(s) {
		s = b.start
	}
	e := a.end
	if b.end.Before(e) {
		e = b.end
	}
	if e.After(s) {
		return interval{start: s, end: e}, true
	}
	return interval{}, false
}

func mergeIntervals(in []interval) []interval {
	if len(in) == 0 {
		return nil
	}
	sort.Slice(in, func(i, j int) bool { return in[i].start.Before(in[j].start) })
	out := []interval{in[0]}
	for _, cur := range in[1:] {
		last := &out[len(out)-1]
		if cur.start.After(last.end) {
			out = append(out, cur)
		} else if cur.end.After(last.end) {
			last.end = cur.end
		}
	}
	return out
}

// OAuth helper ---------------------------------------------

func getClient(ctx context.Context, credentialsPath, tokenPath string, scopes ...string) (oauth2.TokenSource, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials: %w", err)
	}
	config, err := google.ConfigFromJSON(b, scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	// Try load saved token
	var tok *oauth2.Token
	if f, err := os.Open(tokenPath); err == nil {
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&tok); err != nil {
			log.Printf("warning: failed to decode token: %v", err)
			tok = nil
		}
	}

	if tok == nil || !tok.Valid() {
		// Use local redirect server for OAuth flow
		tok = getTokenFromWeb(ctx, config)
		if tok == nil {
			return nil, fmt.Errorf("unable to retrieve token")
		}

		// Save token
		if err := saveToken(tokenPath, tok); err != nil {
			log.Printf("warning: failed to save token: %v", err)
		}
	}

	ts := config.TokenSource(ctx, tok)
	return ts, nil
}

func saveToken(tokenPath string, tok *oauth2.Token) error {
	f, err := os.Create(tokenPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	// Start local server to receive the redirect
	codeCh := make(chan string, 1)
	errorCh := make(chan error, 1)

	// Use localhost with a random available port
	server := &http.Server{
		Addr:              "localhost:0",
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Setup handler for OAuth callback
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errorCh <- fmt.Errorf("no code in callback")
			http.Error(w, "No code found", http.StatusBadRequest)
			return
		}

		// Send success response to browser
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; }
        .success { color: #4CAF50; font-size: 24px; }
    </style>
</head>
<body>
    <div class="success">✓ Authentication successful!</div>
    <p>You can close this window and return to the terminal.</p>
    <script>window.setTimeout(function(){window.close();},3000);</script>
</body>
</html>`)

		codeCh <- code
	})

	// Start server in background
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		log.Fatalf("failed to get TCP address")
	}
	port := tcpAddr.Port

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			errorCh <- serveErr
		}
	}()

	// Update redirect URI to use the actual port
	redirectURL := fmt.Sprintf("http://localhost:%d/callback", port)
	config.RedirectURL = redirectURL

	// Generate auth URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If browser doesn't open automatically, please visit:\n%s\n\n", authURL)

	// Try to open browser automatically
	openBrowser(authURL)

	// Wait for the authorization code or error
	var code string
	select {
	case code = <-codeCh:
		fmt.Println("Authorization code received!")
	case serverErr := <-errorCh:
		log.Fatalf("server error: %v", serverErr)
	case <-time.After(5 * time.Minute):
		log.Fatalf("timeout waiting for authorization")
	}

	// Shutdown the server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
		log.Printf("server shutdown error: %v", shutdownErr)
	}
	cancel()

	// Exchange code for token
	tok, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("unable to retrieve token: %v", err)
	}

	return tok
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Printf("failed to open browser: %v", err)
	}
}

// -----------------------------------------------------------

type config struct {
	credentialsPath string
	tokenPath       string
	calendarID      string
	startStr        string
	endStr          string
	workStart       string
	workEnd         string
	minMinutes      int
	tzName          string
}

func parseFlags() *config {
	c := &config{}
	flag.StringVar(&c.credentialsPath, "credentials", "",
		"Path to OAuth client credentials (credentials.json)")
	flag.StringVar(&c.tokenPath, "token", "token.json", "Path to save/load OAuth token")
	flag.StringVar(&c.calendarID, "calendar", "primary",
		"Calendar ID (e.g., primary or somebody@example.com)")
	flag.StringVar(&c.startStr, "start", "", "Start date (YYYY-MM-DD)")
	flag.StringVar(&c.endStr, "end", "", "End date (YYYY-MM-DD)")
	flag.StringVar(&c.workStart, "workstart", "09:00", "Workday start (HH:MM)")
	flag.StringVar(&c.workEnd, "workend", "17:00", "Workday end (HH:MM)")
	flag.IntVar(&c.minMinutes, "min", 60, "Minimum free slot length in minutes")
	flag.StringVar(&c.tzName, "tz", "Asia/Tokyo", "IANA timezone (e.g., Asia/Tokyo)")
	flag.Parse()

	if c.credentialsPath == "" || c.startStr == "" || c.endStr == "" {
		flag.Usage()
		os.Exit(2)
	}

	return c
}

func fetchCalendarEvents(
	_ context.Context,
	svc *calendar.Service,
	calendarID string,
	startDate, endDate time.Time,
	loc *time.Location,
) ([]*calendar.Event, error) {
	timeMin := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, loc).Format(time.RFC3339)
	timeMax := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, loc).Format(time.RFC3339)

	eventsCall := svc.Events.List(calendarID).
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		OrderBy("startTime").
		ShowDeleted(false)

	events := []*calendar.Event{}
	pageToken := ""
	for {
		if pageToken != "" {
			eventsCall.PageToken(pageToken)
		}
		resp, err := eventsCall.Do()
		if err != nil {
			return nil, err
		}
		events = append(events, resp.Items...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return events, nil
}

func parseEventTime(e *calendar.Event, loc *time.Location) (start, end time.Time, ok bool) {
	if e.Start != nil && e.Start.DateTime != "" && e.End != nil && e.End.DateTime != "" {
		// timed event
		ss, err1 := time.Parse(time.RFC3339, e.Start.DateTime)
		ee, err2 := time.Parse(time.RFC3339, e.End.DateTime)
		if err1 != nil || err2 != nil {
			return time.Time{}, time.Time{}, false
		}
		return ss.In(loc), ee.In(loc), true
	}

	if e.Start != nil && e.Start.Date != "" && e.End != nil && e.End.Date != "" {
		// all-day (dates are in calendar's timezone)
		ds, err1 := time.ParseInLocation("2006-01-02", e.Start.Date, loc)
		de, err2 := time.ParseInLocation("2006-01-02", e.End.Date, loc)
		if err1 != nil || err2 != nil {
			return time.Time{}, time.Time{}, false
		}
		// all-day spans [start 00:00, end 00:00 next-day)
		s := time.Date(ds.Year(), ds.Month(), ds.Day(), 0, 0, 0, 0, loc)
		en := time.Date(de.Year(), de.Month(), de.Day(), 0, 0, 0, 0, loc)
		return s, en, true
	}

	return time.Time{}, time.Time{}, false
}

func eventsToIntervals(events []*calendar.Event, loc *time.Location) []interval {
	busyAll := make([]interval, 0, len(events))
	for _, e := range events {
		if strings.EqualFold(e.Status, "canceled") {
			continue
		}
		if strings.EqualFold(e.Transparency, "transparent") {
			continue // free events
		}

		s, en, ok := parseEventTime(e, loc)
		if !ok {
			continue
		}
		if !en.After(s) {
			continue
		}
		busyAll = append(busyAll, interval{start: s, end: en})
	}
	return busyAll
}

func findFreeSlots(dayStart, dayEnd time.Time, busyAll []interval, minDur time.Duration) []string {
	dayWin := interval{start: dayStart, end: dayEnd}

	// collect and merge overlaps with day window
	var busy []interval
	for _, ev := range busyAll {
		if inter, ok := overlaps(ev, dayWin); ok {
			busy = append(busy, inter)
		}
	}
	busy = mergeIntervals(busy)

	// free slots
	var free []interval
	cursor := dayStart
	for _, b := range busy {
		if b.start.After(cursor) {
			free = append(free, interval{start: cursor, end: b.start})
		}
		if b.end.After(cursor) {
			cursor = b.end
		}
	}
	if cursor.Before(dayEnd) {
		free = append(free, interval{start: cursor, end: dayEnd})
	}

	// filter by min
	var out []string
	for _, f := range free {
		if f.end.Sub(f.start) >= minDur {
			out = append(out, fmt.Sprintf("%02d:%02d~%02d:%02d",
				f.start.Hour(), f.start.Minute(), f.end.Hour(), f.end.Minute()))
		}
	}
	return out
}

// -----------------------------------------------------------

func main() {
	cfg := parseFlags()

	loc, err := time.LoadLocation(cfg.tzName)
	if err != nil {
		log.Fatalf("failed to load timezone %q: %v", cfg.tzName, err)
	}

	startDate, err := time.ParseInLocation("2006-01-02", cfg.startStr, loc)
	if err != nil {
		log.Fatalf("invalid -start: %v", err)
	}
	endDate, err := time.ParseInLocation("2006-01-02", cfg.endStr, loc)
	if err != nil {
		log.Fatalf("invalid -end: %v", err)
	}
	if endDate.Before(startDate) {
		log.Fatalf("-end is before -start")
	}

	wsH, wsM := mustParseClock(cfg.workStart)
	weH, weM := mustParseClock(cfg.workEnd)

	ctx := context.Background()
	ts, err := getClient(ctx, cfg.credentialsPath, cfg.tokenPath, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("unable to get client: %v", err)
	}
	svc, err := calendar.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		log.Fatalf("unable to create calendar service: %v", err)
	}

	// Fetch events
	events, err := fetchCalendarEvents(ctx, svc, cfg.calendarID, startDate, endDate, loc)
	if err != nil {
		log.Fatalf("events list error: %v", err)
	}

	// Convert to intervals
	busyAll := eventsToIntervals(events, loc)

	// Iterate weekdays and print free slots
	minDur := time.Duration(cfg.minMinutes) * time.Minute
	for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}

		dayStart := time.Date(day.Year(), day.Month(), day.Day(), wsH, wsM, 0, 0, loc)
		dayEnd := time.Date(day.Year(), day.Month(), day.Day(), weH, weM, 0, 0, loc)

		out := findFreeSlots(dayStart, dayEnd, busyAll, minDur)
		if len(out) == 0 {
			continue
		}
		fmt.Printf("- %s（%s） %s\n", day.Format("2006-01-02"), formatJpWeekday(day), strings.Join(out, ", "))
	}
}
