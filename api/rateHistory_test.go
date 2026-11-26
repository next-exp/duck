package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/jmbenlloch/next_duck/rpc/api/apiconnect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func historyOrigin() time.Time    { return time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC) }
func closeRate(a, b float64) bool { return math.Abs(a-b) < 1e-8 }

func TestRateHistoryWeightedCounterRebin(t *testing.T) {
	origin := historyOrigin()
	// A zoom cuts both a low-rate interval and a high-rate interval in half.
	bins := newHistoryBins(origin, origin.Add(2*time.Second), origin.Add(8*time.Second), 10*time.Second)
	addHistoryInterval(bins, historySample{at: origin, events: 100, bytes: 1000}, historySample{at: origin.Add(5 * time.Second), events: 150, bytes: 1500}, 15*time.Second, 10*time.Second)
	addHistoryInterval(bins, historySample{at: origin.Add(5 * time.Second), events: 150, bytes: 1500}, historySample{at: origin.Add(10 * time.Second), events: 250, bytes: 2500}, 15*time.Second, 10*time.Second)
	points := historyResponseBins(bins, 1)
	if len(points) != 1 || !points[0].Complete || !points[0].Partial || !closeRate(points[0].GetTriggerRateHz(), 15) || !closeRate(points[0].GetByteRate(), 150) {
		t.Fatalf("wrong weighted zoom result: %v", points)
	}
}

func TestRateHistoryGDCsumAndCoverage(t *testing.T) {
	origin := historyOrigin()
	bins := newHistoryBins(origin, origin, origin.Add(10*time.Second), 5*time.Second)
	// Asynchronous GDC samples can cover the same bins without matching timestamps.
	addHistoryInterval(bins, historySample{at: origin.Add(-time.Second), events: 100}, historySample{at: origin.Add(11 * time.Second), events: 340}, 15*time.Second, 5*time.Second)
	addHistoryInterval(bins, historySample{at: origin, events: 100}, historySample{at: origin.Add(5 * time.Second), events: 250}, 15*time.Second, 5*time.Second)
	points := historyResponseBins(bins, 2)
	if !points[0].Complete || !closeRate(points[0].GetTriggerRateHz(), 50) {
		t.Fatalf("GDCs not summed correctly: %v", points[0])
	}
	if points[1].Complete || points[1].TriggerRateHz != nil || points[1].ByteRate != nil || !closeRate(points[1].Coverage, 0.5) {
		t.Fatalf("missing host shown as a rate: %v", points[1])
	}
}

func TestRateHistoryResetGapAndZero(t *testing.T) {
	origin := historyOrigin()
	for _, tc := range []struct {
		name  string
		end   historySample
		gap   time.Duration
		valid bool
	}{
		{"zero rate", historySample{at: origin.Add(5 * time.Second), events: 100, bytes: 1000}, 15 * time.Second, true},
		{"event reset", historySample{at: origin.Add(5 * time.Second), events: 1, bytes: 1000}, 15 * time.Second, false},
		{"byte reset", historySample{at: origin.Add(5 * time.Second), events: 110, bytes: 1}, 15 * time.Second, false},
		{"explicit reset", historySample{at: origin.Add(5 * time.Second), events: 110, bytes: 1100, reset: true}, 15 * time.Second, false},
		{"long telemetry gap", historySample{at: origin.Add(20 * time.Second), events: 110, bytes: 1100}, 15 * time.Second, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bins := newHistoryBins(origin, origin, origin.Add(5*time.Second), 5*time.Second)
			addHistoryInterval(bins, historySample{at: origin, events: 100, bytes: 1000}, tc.end, tc.gap, 5*time.Second)
			point := historyResponseBins(bins, 1)[0]
			if point.Complete != tc.valid {
				t.Fatalf("wrong coverage: %v", point)
			}
			if tc.valid && (point.TriggerRateHz == nil || *point.TriggerRateHz != 0) {
				t.Fatal("real zero must be present")
			}
			if !tc.valid && point.TriggerRateHz != nil {
				t.Fatal("gap must have an absent rate")
			}
		})
	}
	// Subtract integer counters before converting to float; large totals lose no delta.
	bins := newHistoryBins(origin, origin, origin.Add(5*time.Second), 5*time.Second)
	addHistoryInterval(bins, historySample{at: origin, events: 1 << 60}, historySample{at: origin.Add(5 * time.Second), events: (1 << 60) + 10}, 15*time.Second, 5*time.Second)
	if !closeRate(historyResponseBins(bins, 1)[0].GetTriggerRateHz(), 2) {
		t.Fatal("large counters lost precision")
	}
}

func TestRateHistoryBinBudgetAndAlignment(t *testing.T) {
	origin := historyOrigin()
	for _, duration := range []time.Duration{time.Second, 30 * time.Minute, 2 * time.Hour, 24 * time.Hour, 7 * 24 * time.Hour, 100 * 365 * 24 * time.Hour} {
		for _, offset := range []time.Duration{0, 1234 * time.Millisecond, 61 * time.Second} {
			for _, budget := range []int{1, 2, 10, 1000, 2000} {
				from, to := origin.Add(offset), origin.Add(offset+duration)
				width := historyBinWidth(origin, from, to, budget)
				bins := newHistoryBins(origin, from, to, width)
				if len(bins) > budget || len(bins) == 0 || !bins[0].start.Equal(from) || !bins[len(bins)-1].end.Equal(to) {
					t.Fatalf("unbounded/misclipped bins: duration=%v offset=%v budget=%d width=%v count=%d", duration, offset, budget, width, len(bins))
				}
				if width/time.Second > math.MaxUint32 {
					t.Fatal("bin width exceeds wire format")
				}
			}
		}
	}
	if width := historyBinWidth(origin, origin, origin.Add(30*time.Minute), 1000); width != 5*time.Second {
		t.Fatalf("short run lost native resolution: %v", width)
	}
}

func TestRateHistoryRequestValidation(t *testing.T) {
	for _, req := range []*pb.GetRateHistoryRequest{
		{}, {Run: -1}, {Run: 1, MaxPoints: 2001}, {Run: 1, MaxGapSeconds: 3601},
		{Run: 1, From: &timestamppb.Timestamp{Seconds: 253402300800}},
		{Run: 1, From: timestamppb.New(historyOrigin()), To: timestamppb.New(historyOrigin())},
	} {
		_, _, err := validateHistoryRequest(req)
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("accepted invalid request %v: %v", req, err)
		}
	}
	points, gap, err := validateHistoryRequest(&pb.GetRateHistoryRequest{Run: 1})
	if err != nil || points != 1000 || gap != 15*time.Second {
		t.Fatal("bad defaults")
	}
}

func TestRateHistoryEmptyAndUnknownRun(t *testing.T) {
	for _, exists := range []bool{true, false} {
		db, mock, _ := sqlmock.New()
		mock.ExpectBegin()
		rows := sqlmock.NewRows([]string{"start", "stop"})
		if exists {
			rows.AddRow(nil, nil)
		}
		mock.ExpectQuery("SELECT start, stop FROM runs").WithArgs(int32(123)).WillReturnRows(rows)
		if exists {
			mock.ExpectQuery("SELECT gdc, MIN").WithArgs(int32(123)).WillReturnRows(sqlmock.NewRows([]string{"gdc", "first", "last"}))
			mock.ExpectCommit()
		} else {
			mock.ExpectRollback()
		}
		response, err := (&DuckAPIServer{historyDB: db}).GetRateHistory(context.Background(), connect.NewRequest(&pb.GetRateHistoryRequest{Run: 123}))
		if exists {
			if err != nil || response.Msg.HasData || len(response.Msg.Bins) != 0 {
				t.Fatalf("bad empty history response: %v %v", response, err)
			}
		} else if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("wrong missing run error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		db.Close()
	}
}

// A disposable MySQL schema is required. This test creates/drops runs and rateSamples.
func TestRateHistoryMySQL(t *testing.T) {
	dsn := os.Getenv("RATE_HISTORY_TEST_DSN")
	if dsn == "" {
		t.Skip("RATE_HISTORY_TEST_DSN not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ddl, err := os.ReadFile("../database/migrations/20261104_add_rate_samples.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(ddl)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE runs (id INT PRIMARY KEY, start TIMESTAMP NULL, stop TIMESTAMP NULL)"); err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP TABLE runs, rateSamples")
	origin := historyOrigin()
	if _, err = db.Exec("INSERT INTO runs VALUES (1,?,NULL),(2,?,NULL)", origin, origin); err != nil {
		t.Fatal(err)
	}
	for _, source := range []struct {
		name            string
		offset          time.Duration
		eventsPerSample int
	}{{"gdc1", 0, 50}, {"gdc2", time.Second, 100}} {
		for i := 0; i <= 6; i++ {
			at := origin.Add(time.Duration(i)*5*time.Second + source.offset)
			// Current rates deliberately disagree with counters: history must ignore them.
			_, err = db.Exec(`INSERT INTO rateSamples VALUES (1,?,?,?, ?,9999,9999,9999,9999,FALSE)`, source.name, at, 1000+i*source.eventsPerSample, 10000+i*source.eventsPerSample*10)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Noise from another run must never influence the roster, counters or bounds.
	if _, err = db.Exec(`INSERT INTO rateSamples VALUES (2,'unrelated',?,0,0,0,0,0,0,FALSE)`, origin); err != nil {
		t.Fatal(err)
	}
	server := NewDuckAPIServer(db, "", "", false)
	path, handler := apiconnect.NewDuckAPIHandler(server)
	_ = path
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	client := apiconnect.NewDuckAPIClient(httpServer.Client(), httpServer.URL)
	request := &pb.GetRateHistoryRequest{Run: 1, From: timestamppb.New(origin.Add(2 * time.Second)), To: timestamppb.New(origin.Add(28 * time.Second)), MaxPoints: 3}
	response, err := client.GetRateHistory(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	result := response.Msg
	if len(result.Bins) > 3 || len(result.Bins) == 0 || len(result.Gdcs) != 2 || result.CoverageBasis != "observed_run_history" {
		t.Fatalf("wrong bounded response: %v", result)
	}
	for _, bin := range result.Bins {
		if !bin.Complete || !closeRate(bin.GetTriggerRateHz(), 30) || !closeRate(bin.GetByteRate(), 300) {
			t.Fatalf("edge predecessor/successor or aggregate wrong: %v", bin)
		}
	}
	// Whole-run auto mode uses native resolution until the budget requires rebinning.
	auto, err := client.GetRateHistory(context.Background(), connect.NewRequest(&pb.GetRateHistoryRequest{Run: 1}))
	if err != nil || auto.Msg.BinWidthSeconds != 5 || len(auto.Msg.Bins) != 7 {
		t.Fatalf("bad auto response: %v %v", auto, err)
	}
	if auto.Msg.Bins[0].TriggerRateHz != nil || auto.Msg.Bins[len(auto.Msg.Bins)-1].TriggerRateHz != nil {
		t.Fatal("incomplete edge bins must not masquerade as zero")
	}
	// JSON optional rates distinguish a healthy zero from missing coverage.
	encoded, err := protojson.Marshal(auto.Msg)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err = json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "9999") {
		t.Fatal("used current rates instead of counters")
	}
	// Explicit windows wholly outside recorded data return no bins.
	outside, err := client.GetRateHistory(context.Background(), connect.NewRequest(&pb.GetRateHistoryRequest{Run: 1, From: timestamppb.New(origin.Add(time.Hour)), To: timestamppb.New(origin.Add(2 * time.Hour))}))
	if err != nil || len(outside.Msg.Bins) != 0 {
		t.Fatal("outside window was extrapolated")
	}
	// A counter reset creates an internal hole, while following intervals recover.
	if _, err = db.Exec("UPDATE rateSamples SET counter_reset=TRUE WHERE run=1 AND gdc='gdc1' AND sample_time=?", origin.Add(15*time.Second)); err != nil {
		t.Fatal(err)
	}
	reset, err := client.GetRateHistory(context.Background(), connect.NewRequest(&pb.GetRateHistoryRequest{Run: 1}))
	if err != nil || reset.Msg.Bins[2].Complete || reset.Msg.Bins[2].TriggerRateHz != nil || !reset.Msg.Bins[3].Complete {
		t.Fatalf("reset handling failed: %v %v", reset, err)
	}
	// Check the range query's execution plan uses the per-run/GDC primary index.
	rows, err := db.Query(`EXPLAIN FORMAT=TRADITIONAL SELECT sample_time,events,bytes,counter_reset FROM rateSamples WHERE run=1 AND gdc='gdc1' AND sample_time>=? AND sample_time<=? ORDER BY sample_time`, origin.Add(5*time.Second), origin.Add(15*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, _ := rows.Columns()
	values := make([]sql.RawBytes, len(columns))
	ptrs := make([]any, len(columns))
	for i := range ptrs {
		ptrs[i] = &values[i]
	}
	if !rows.Next() {
		t.Fatal("missing query plan")
	}
	if err := rows.Scan(ptrs...); err != nil {
		t.Fatal(err)
	}
	indexed := false
	for i, name := range columns {
		if name == "key" && string(values[i]) == "PRIMARY" {
			indexed = true
		}
	}
	if !indexed {
		t.Fatalf("history range read did not use primary key: %q", values)
	}
}

func BenchmarkRateHistoryRebin(b *testing.B) {
	origin := historyOrigin()
	const samples = 120000
	end := origin.Add(samples * 5 * time.Second)
	width := historyBinWidth(origin, origin, end, 1000)
	for i := 0; i < b.N; i++ {
		bins := newHistoryBins(origin, origin, end, width)
		for host := 0; host < 7; host++ {
			for j := 0; j < samples; j++ {
				previous := historySample{at: origin.Add(time.Duration(j) * 5 * time.Second), events: uint64(j * 100)}
				current := historySample{at: previous.at.Add(5 * time.Second), events: uint64((j + 1) * 100)}
				addHistoryInterval(bins, previous, current, 15*time.Second, width)
			}
		}
		if len(historyResponseBins(bins, 7)) > 1000 {
			b.Fatal(fmt.Errorf("point budget exceeded"))
		}
	}
}

func TestRateHistoryDatabaseErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure error
		code    connect.Code
	}{
		{"missing migration", &mysql.MySQLError{Number: 1146, Message: "table missing"}, connect.CodeFailedPrecondition},
		{"deadline", context.DeadlineExceeded, connect.CodeDeadlineExceeded},
		{"canceled", context.Canceled, connect.CodeCanceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT start, stop FROM runs").WithArgs(int32(1)).WillReturnRows(sqlmock.NewRows([]string{"start", "stop"}).AddRow(nil, nil))
			mock.ExpectQuery("SELECT gdc, MIN").WithArgs(int32(1)).WillReturnError(tc.failure)
			mock.ExpectRollback()
			_, err := (&DuckAPIServer{historyDB: db}).GetRateHistory(context.Background(), connect.NewRequest(&pb.GetRateHistoryRequest{Run: 1}))
			if connect.CodeOf(err) != tc.code {
				t.Fatalf("wrong error code: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRateHistoryOptionalZeroOnWire(t *testing.T) {
	origin := historyOrigin()
	bins := newHistoryBins(origin, origin, origin.Add(5*time.Second), 5*time.Second)
	addHistoryInterval(bins, historySample{at: origin, events: 1}, historySample{at: origin.Add(5 * time.Second), events: 1}, 15*time.Second, 5*time.Second)
	point := historyResponseBins(bins, 1)[0]
	encoded, err := protojson.Marshal(point)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if value, present := wire["triggerRateHz"]; !present || value != float64(0) {
		t.Fatalf("zero rate absent from JSON: %s", encoded)
	}
	point = historyResponseBins(bins, 2)[0]
	encoded, err = protojson.Marshal(point)
	if err != nil {
		t.Fatal(err)
	}
	wire = make(map[string]any)
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if _, present := wire["triggerRateHz"]; present {
		t.Fatalf("incomplete rate present in JSON: %s", encoded)
	}
}
