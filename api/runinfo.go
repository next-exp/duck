package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmbenlloch/next_duck/pkg/database"
)

func getRun(queries database.Querier) (database.Run, error) {
	return queries.GetLatestRunWithTimestamp(context.Background())
}

func addRun(queries database.Querier) error {
	result, err := queries.GetLatestRunWithTimestamp(context.Background())
	if err != nil {
		return err
	}

	if result.Start.Valid {
		err = queries.InsertNewRun(context.Background())
	}
	return err
}

func updateRunStartTime(queries database.Querier, run int) error {
	params := database.UpdateRunStartTimeParams{
		Start: sql.NullTime{Time: time.Now(), Valid: true},
		ID:    int32(run),
	}
	return queries.UpdateRunStartTime(context.Background(), params)
}

func updateRunStopTime(queries database.Querier, run int) error {
	params := database.UpdateRunStopTimeParams{
		Stop: sql.NullTime{Time: time.Now(), Valid: true},
		ID:   int32(run),
	}
	return queries.UpdateRunStopTime(context.Background(), params)
}

func insertGDCEvents(queries database.Querier, run int, gdc int, events int64) error {
	params := database.InsertGDCEventsParams{
		Run:    int32(run),
		GdcID:  sql.NullInt32{Int32: int32(gdc), Valid: true},
		Events: sql.NullInt64{Int64: events, Valid: true},
	}
	return queries.InsertGDCEvents(context.Background(), params)
}

func insertGDCBytes(queries database.Querier, run int, gdc int, bytes int64) error {
	params := database.InsertGDCBytesParams{
		Run:   int32(run),
		GdcID: sql.NullInt32{Int32: int32(gdc), Valid: true},
		Bytes: sql.NullInt64{Int64: bytes, Valid: true},
	}
	return queries.InsertGDCBytes(context.Background(), params)
}

func insertLDCEvents(queries database.Querier, run int, ldc int, events int64) error {
	params := database.InsertLDCEventsParams{
		Run:    int32(run),
		LdcID:  sql.NullInt32{Int32: int32(ldc), Valid: true},
		Events: sql.NullInt64{Int64: events, Valid: true},
	}
	return queries.InsertLDCEvents(context.Background(), params)
}

func insertLDCBytes(queries database.Querier, run int, ldc int, bytes int64) error {
	params := database.InsertLDCBytesParams{
		Run:   int32(run),
		LdcID: sql.NullInt32{Int32: int32(ldc), Valid: true},
		Bytes: sql.NullInt64{Int64: bytes, Valid: true},
	}
	return queries.InsertLDCBytes(context.Background(), params)
}

func insertLDCErrorCount(queries database.Querier, run int, ldc int, nErrors int64) error {
	params := database.InsertLDCErrorCountParams{
		Run:    int32(run),
		LdcID:  sql.NullInt32{Int32: int32(ldc), Valid: true},
		Errors: sql.NullInt64{Int64: nErrors, Valid: true},
	}
	return queries.InsertLDCErrorCount(context.Background(), params)
}

func insertGDCErrorCount(queries database.Querier, run int, gdc int, nErrors int64) error {
	params := database.InsertGDCErrorCountParams{
		Run:    int32(run),
		GdcID:  sql.NullInt32{Int32: int32(gdc), Valid: true},
		Errors: sql.NullInt64{Int64: nErrors, Valid: true},
	}
	return queries.InsertGDCErrorCount(context.Background(), params)
}
