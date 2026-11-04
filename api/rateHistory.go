package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"connectrpc.com/connect"
	"github.com/go-sql-driver/mysql"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultHistoryPoints = 1000
	maximumHistoryPoints = 2000
	defaultHistoryGap    = 15
	maximumHistoryGap    = 3600
	maximumHistoryGDCs   = 256
)

type historySource struct {
	name        string
	first, last time.Time
}

type historySample struct {
	at            time.Time
	events, bytes uint64
	reset         bool
}

type historyBin struct {
	start, end                    time.Time
	events, bytes, coveredSeconds float64
	partial                       bool
}

func validateHistoryRequest(req *pb.GetRateHistoryRequest) (int, time.Duration, error) {
	invalid := func(message string) (int, time.Duration, error) {
		return 0, 0, connect.NewError(connect.CodeInvalidArgument, errors.New(message))
	}
	if req.Run <= 0 {
		return invalid("run must be positive")
	}
	for _, t := range []*timestamppb.Timestamp{req.From, req.To} {
		if t != nil && t.CheckValid() != nil {
			return invalid("invalid history timestamp")
		}
	}
	if req.From != nil && req.To != nil && !req.From.AsTime().Before(req.To.AsTime()) {
		return invalid("from must precede to")
	}
	points := int(req.MaxPoints)
	if points == 0 {
		points = defaultHistoryPoints
	}
	if req.MaxPoints > maximumHistoryPoints {
		return invalid("max_points must be between 1 and 2000 (or 0 for default)")
	}
	gap := req.MaxGapSeconds
	if gap == 0 {
		gap = defaultHistoryGap
	}
	if gap > maximumHistoryGap {
		return invalid("max_gap_seconds must not exceed 3600")
	}
	return points, time.Duration(gap) * time.Second, nil
}

// Choose stable run-start-aligned bins; the two clipped edge bins count against
// the budget too. This avoids a max_points+1 response for unaligned zoom windows.
func historyBinWidth(origin, from, to time.Time, maxPoints int) time.Duration {
	widths := []int64{5, 10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 21600, 43200, 86400}
	fits := func(width time.Duration) bool {
		first := from.Sub(origin) / width
		last := to.Sub(origin) / width
		if to.Sub(origin)%width != 0 {
			last++
		}
		return last-first <= time.Duration(maxPoints)
	}
	for _, seconds := range widths {
		width := time.Duration(seconds) * time.Second
		if fits(width) {
			return width
		}
	}
	width := 24 * time.Hour
	for !fits(width) {
		width *= 2
		if width > time.Duration(math.MaxUint32)*time.Second {
			return time.Duration(math.MaxUint32) * time.Second
		}
	}
	return width
}

func newHistoryBins(origin, from, to time.Time, width time.Duration) []historyBin {
	bins := make([]historyBin, 0)
	for start := origin.Add((from.Sub(origin) / width) * width); start.Before(to); start = start.Add(width) {
		end := start.Add(width)
		clippedStart, clippedEnd := start, end
		if clippedStart.Before(from) {
			clippedStart = from
		}
		if clippedEnd.After(to) {
			clippedEnd = to
		}
		bins = append(bins, historyBin{start: clippedStart, end: clippedEnd, partial: !clippedStart.Equal(start) || !clippedEnd.Equal(end)})
	}
	return bins
}

func addHistoryInterval(bins []historyBin, previous, current historySample, gap time.Duration, width time.Duration) {
	elapsed := current.at.Sub(previous.at)
	if elapsed <= 0 || elapsed > gap || current.reset || current.events < previous.events || current.bytes < previous.bytes || len(bins) == 0 {
		return
	}
	start, end := previous.at, current.at
	if start.Before(bins[0].start) {
		start = bins[0].start
	}
	if end.After(bins[len(bins)-1].end) {
		end = bins[len(bins)-1].end
	}
	if !start.Before(end) {
		return
	}
	// Account for a potentially clipped first bin. The loop visits only bins
	// intersected by this interval, not all bins for every source sample.
	i := 0
	if !start.Before(bins[0].end) {
		i = 1 + int(start.Sub(bins[0].end)/width)
	}
	eventRate := float64(current.events-previous.events) / elapsed.Seconds()
	byteRate := float64(current.bytes-previous.bytes) / elapsed.Seconds()
	for ; i < len(bins) && bins[i].start.Before(end); i++ {
		left, right := start, end
		if left.Before(bins[i].start) {
			left = bins[i].start
		}
		if right.After(bins[i].end) {
			right = bins[i].end
		}
		seconds := right.Sub(left).Seconds()
		if seconds <= 0 {
			continue
		}
		bins[i].events += eventRate * seconds
		bins[i].bytes += byteRate * seconds
		bins[i].coveredSeconds += seconds
	}
}

func historyResponseBins(bins []historyBin, sources int) []*pb.RateHistoryBin {
	result := make([]*pb.RateHistoryBin, 0, len(bins))
	for _, bin := range bins {
		duration := bin.end.Sub(bin.start).Seconds()
		coverage := math.Min(1, bin.coveredSeconds/(duration*float64(sources)))
		complete := coverage >= 1-1e-9
		point := &pb.RateHistoryBin{Start: timestamppb.New(bin.start), End: timestamppb.New(bin.end), Coverage: coverage, Complete: complete, Partial: bin.partial}
		if complete {
			point.TriggerRateHz = proto.Float64(bin.events / duration)
			point.ByteRate = proto.Float64(bin.bytes / duration)
		}
		result = append(result, point)
	}
	return result
}

func historyError(err error) error {
	if errors.Is(err, context.Canceled) {
		return connect.NewError(connect.CodeCanceled, errors.New("history request canceled"))
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return connect.NewError(connect.CodeDeadlineExceeded, errors.New("history query timed out; select a smaller window"))
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("rate history table unavailable; apply the rateSamples migration"))
	}
	log.Printf("rate history query failed: %v", err)
	return connect.NewError(connect.CodeInternal, errors.New("could not read rate history"))
}

// GetRateHistory serves both whole-run polling and fixed zoom windows. It never
// interpolates across a reset or a telemetry gap longer than max_gap_seconds.
func (s *DuckAPIServer) GetRateHistory(ctx context.Context, req *connect.Request[pb.GetRateHistoryRequest]) (*connect.Response[pb.GetRateHistoryResponse], error) {
	maxPoints, maxGap, err := validateHistoryRequest(req.Msg)
	if err != nil {
		return nil, err
	}
	if s.historyDB == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("history database unavailable"))
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Metadata and source samples share a consistent snapshot while rateReader
	// continues inserting data. Only the API process's memory is bounded here;
	// query execution still scales with the selected run/window.
	tx, err := s.historyDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, historyError(err)
	}
	defer tx.Rollback()
	var start, stop sql.NullTime
	if err := tx.QueryRowContext(ctx, "SELECT start, stop FROM runs WHERE id = ?", req.Msg.Run).Scan(&start, &stop); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("run not found"))
		}
		return nil, historyError(err)
	}
	sources, err := loadHistorySources(ctx, tx, req.Msg.Run)
	if err != nil {
		return nil, historyError(err)
	}
	if len(sources) > maximumHistoryGDCs {
		return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("too many GDCs in rate history"))
	}
	response := &pb.GetRateHistoryResponse{Run: req.Msg.Run, CoverageBasis: "observed_run_history", MaxGapSeconds: uint32(maxGap / time.Second), Bins: []*pb.RateHistoryBin{}, Gdcs: []string{}}
	if start.Valid {
		response.RunStart = timestamppb.New(start.Time.UTC())
	}
	if stop.Valid {
		response.RunStop = timestamppb.New(stop.Time.UTC())
	}
	if len(sources) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, historyError(err)
		}
		return connect.NewResponse(response), nil
	}
	earliest, latest := sources[0].first, sources[0].last
	for _, source := range sources {
		response.Gdcs = append(response.Gdcs, source.name)
		if source.first.Before(earliest) {
			earliest = source.first
		}
		if source.last.After(latest) {
			latest = source.last
		}
	}
	response.HasData = true
	response.LatestSample = timestamppb.New(latest)
	origin := earliest
	if start.Valid {
		origin = start.Time.UTC()
	}
	end := latest
	if stop.Valid && stop.Time.Before(end) {
		end = stop.Time.UTC()
	}
	if end.Before(origin) {
		end = origin
	}
	from, to := origin, end
	clamp := func(t time.Time) time.Time {
		if t.Before(origin) {
			return origin
		}
		if t.After(end) {
			return end
		}
		return t.UTC()
	}
	if req.Msg.From != nil {
		from = clamp(req.Msg.From.AsTime())
	}
	if req.Msg.To != nil {
		to = clamp(req.Msg.To.AsTime())
	}
	response.From, response.To = timestamppb.New(from), timestamppb.New(to)
	if !from.Before(to) {
		if err := tx.Commit(); err != nil {
			return nil, historyError(err)
		}
		return connect.NewResponse(response), nil
	}
	// Reject corrupt/implausible timestamps before duration arithmetic can overflow.
	if to.Sub(origin) > 100*365*24*time.Hour {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rate history exceeds the supported 100-year span"))
	}
	width := historyBinWidth(origin, from, to, maxPoints)
	bins := newHistoryBins(origin, from, to, width)
	for _, source := range sources {
		if err := accumulateHistorySource(ctx, tx, req.Msg.Run, source.name, from, to, maxGap, width, bins); err != nil {
			return nil, historyError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, historyError(err)
	}
	response.BinWidthSeconds = uint32(width / time.Second)
	response.Bins = historyResponseBins(bins, len(sources))
	return connect.NewResponse(response), nil
}

func loadHistorySources(ctx context.Context, tx *sql.Tx, run int32) ([]historySource, error) {
	// The primary key (run,gdc,sample_time) supports this recorded-run roster.
	rows, err := tx.QueryContext(ctx, `SELECT gdc, MIN(sample_time), MAX(sample_time)
FROM rateSamples WHERE run = ? GROUP BY gdc ORDER BY gdc LIMIT 257`, run)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sources := make([]historySource, 0)
	for rows.Next() {
		var source historySource
		if err := rows.Scan(&source.name, &source.first, &source.last); err != nil {
			return nil, err
		}
		source.first, source.last = source.first.UTC(), source.last.UTC()
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func accumulateHistorySource(ctx context.Context, tx *sql.Tx, run int32, gdc string, from, to time.Time, maxGap, width time.Duration, bins []historyBin) error {
	// Include one predecessor and one successor. Both scalar boundary lookups and
	// the ordered stream use the (run,gdc,sample_time) primary key. Samples outside
	// the requested window never appear in the response; they define edge deltas.
	rows, err := tx.QueryContext(ctx, `SELECT sample_time, events, bytes, counter_reset
FROM rateSamples WHERE run = ? AND gdc = ?
AND sample_time >= COALESCE((SELECT MAX(sample_time) FROM rateSamples WHERE run = ? AND gdc = ? AND sample_time <= ?), ?)
AND sample_time <= COALESCE((SELECT MIN(sample_time) FROM rateSamples WHERE run = ? AND gdc = ? AND sample_time >= ?), ?)
ORDER BY sample_time`, run, gdc, run, gdc, from, from, run, gdc, to, to)
	if err != nil {
		return err
	}
	defer rows.Close()
	var previous historySample
	havePrevious := false
	for rows.Next() {
		var current historySample
		if err := rows.Scan(&current.at, &current.events, &current.bytes, &current.reset); err != nil {
			return err
		}
		current.at = current.at.UTC()
		if havePrevious {
			addHistoryInterval(bins, previous, current, maxGap, width)
		}
		previous, havePrevious = current, true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("streaming GDC rate history: %w", err)
	}
	return nil
}
