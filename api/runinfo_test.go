package main

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetRun_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedRun := int32(123)
	expectedStart := time.Now()
	expectedStop := time.Now().Add(time.Hour)

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    expectedRun,
		Start: sql.NullTime{Time: expectedStart, Valid: true},
		Stop:  sql.NullTime{Time: expectedStop, Valid: true},
	}, nil)

	result, err := getRun(mockQuerier)

	require.NoError(t, err)
	assert.Equal(t, expectedRun, result.ID)
	assert.True(t, result.Start.Valid)
	assert.True(t, result.Stop.Valid)
	assert.Equal(t, expectedStart.Unix(), result.Start.Time.Unix())
	assert.Equal(t, expectedStop.Unix(), result.Stop.Time.Unix())
	mockQuerier.AssertExpectations(t)
}

func TestGetRun_WithNullStopTime(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedRun := int32(123)
	expectedStart := time.Now()

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    expectedRun,
		Start: sql.NullTime{Time: expectedStart, Valid: true},
		Stop:  sql.NullTime{Valid: false},
	}, nil)

	result, err := getRun(mockQuerier)

	require.NoError(t, err)
	assert.Equal(t, expectedRun, result.ID)
	assert.True(t, result.Start.Valid)
	assert.False(t, result.Stop.Valid, "Stop time should be null for active run")
	mockQuerier.AssertExpectations(t)
}

func TestGetRun_WithNullStartTime(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedRun := int32(124)

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    expectedRun,
		Start: sql.NullTime{Valid: false},
		Stop:  sql.NullTime{Valid: false},
	}, nil)

	result, err := getRun(mockQuerier)

	require.NoError(t, err)
	assert.Equal(t, expectedRun, result.ID)
	assert.False(t, result.Start.Valid, "Start time should be null for unstarted run")
	assert.False(t, result.Stop.Valid)
	mockQuerier.AssertExpectations(t)
}

func TestGetRun_NoRows(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).
		Return(database.Run{}, sql.ErrNoRows)

	_, err := getRun(mockQuerier)

	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateRunStartTime_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	runNumber := 123

	mockQuerier.On("UpdateRunStartTime", mock.Anything, mock.MatchedBy(func(arg database.UpdateRunStartTimeParams) bool {
		return arg.ID == int32(runNumber) && arg.Start.Valid
	})).Return(nil)

	err := updateRunStartTime(mockQuerier, runNumber)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateRunStartTime_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	runNumber := 123

	mockQuerier.On("UpdateRunStartTime", mock.Anything, mock.Anything).
		Return(sql.ErrConnDone)

	err := updateRunStartTime(mockQuerier, runNumber)

	assert.Error(t, err)
	assert.Equal(t, sql.ErrConnDone, err)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateRunStopTime_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	runNumber := 123

	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.MatchedBy(func(arg database.UpdateRunStopTimeParams) bool {
		return arg.ID == int32(runNumber) && arg.Stop.Valid
	})).Return(nil)

	err := updateRunStopTime(mockQuerier, runNumber)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateRunStopTime_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	runNumber := 123

	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.Anything).
		Return(sql.ErrConnDone)

	err := updateRunStopTime(mockQuerier, runNumber)

	assert.Error(t, err)
	assert.Equal(t, sql.ErrConnDone, err)
	mockQuerier.AssertExpectations(t)
}

func TestAddRun_WhenPreviousRunStarted(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedRun := int32(123)
	expectedStart := time.Now()

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    expectedRun,
		Start: sql.NullTime{Time: expectedStart, Valid: true},
		Stop:  sql.NullTime{Valid: false},
	}, nil)

	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	err := addRun(mockQuerier)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestAddRun_WhenPreviousRunNotStarted(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedRun := int32(123)

	// Previous run has no start time, so InsertNewRun should NOT be called
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    expectedRun,
		Start: sql.NullTime{Valid: false},
		Stop:  sql.NullTime{Valid: false},
	}, nil)

	err := addRun(mockQuerier)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
	// Verify InsertNewRun was NOT called
	mockQuerier.AssertNotCalled(t, "InsertNewRun", mock.Anything)
}

func TestAddRun_QueryError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).
		Return(database.Run{}, sql.ErrConnDone)

	err := addRun(mockQuerier)

	assert.Error(t, err)
	assert.Equal(t, sql.ErrConnDone, err)
	mockQuerier.AssertExpectations(t)
}

func TestInsertGDCEvents_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	run := 123
	gdcID := 1
	events := int64(1000)

	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.MatchedBy(func(arg database.InsertGDCEventsParams) bool {
		return arg.Run == int32(run) &&
			arg.GdcID.Int32 == int32(gdcID) &&
			arg.Events.Int64 == events
	})).Return(nil)

	err := insertGDCEvents(mockQuerier, run, gdcID, events)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestInsertGDCEvents_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.Anything).
		Return(fmt.Errorf("database error"))

	err := insertGDCEvents(mockQuerier, 123, 1, 1000)

	assert.Error(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestInsertGDCBytes_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	run := 123
	gdcID := 1
	bytes := int64(1024000)

	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.MatchedBy(func(arg database.InsertGDCBytesParams) bool {
		return arg.Run == int32(run) &&
			arg.GdcID.Int32 == int32(gdcID) &&
			arg.Bytes.Int64 == bytes
	})).Return(nil)

	err := insertGDCBytes(mockQuerier, run, gdcID, bytes)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestInsertGDCErrorCount_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	run := 123
	gdcID := 1
	errors := int64(5)

	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.MatchedBy(func(arg database.InsertGDCErrorCountParams) bool {
		return arg.Run == int32(run) &&
			arg.GdcID.Int32 == int32(gdcID) &&
			arg.Errors.Int64 == errors
	})).Return(nil)

	err := insertGDCErrorCount(mockQuerier, run, gdcID, errors)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestInsertLDCEvents_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	run := 123
	ldcID := 1
	events := int64(1000)

	mockQuerier.On("InsertLDCEvents", mock.Anything, mock.MatchedBy(func(arg database.InsertLDCEventsParams) bool {
		return arg.Run == int32(run) &&
			arg.LdcID.Int32 == int32(ldcID) &&
			arg.Events.Int64 == events
	})).Return(nil)

	err := insertLDCEvents(mockQuerier, run, ldcID, events)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestInsertLDCBytes_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	run := 123
	ldcID := 1
	bytes := int64(1024000)

	mockQuerier.On("InsertLDCBytes", mock.Anything, mock.MatchedBy(func(arg database.InsertLDCBytesParams) bool {
		return arg.Run == int32(run) &&
			arg.LdcID.Int32 == int32(ldcID) &&
			arg.Bytes.Int64 == bytes
	})).Return(nil)

	err := insertLDCBytes(mockQuerier, run, ldcID, bytes)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestInsertLDCErrorCount_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	run := 123
	ldcID := 1
	errors := int64(5)

	mockQuerier.On("InsertLDCErrorCount", mock.Anything, mock.MatchedBy(func(arg database.InsertLDCErrorCountParams) bool {
		return arg.Run == int32(run) &&
			arg.LdcID.Int32 == int32(ldcID) &&
			arg.Errors.Int64 == errors
	})).Return(nil)

	err := insertLDCErrorCount(mockQuerier, run, ldcID, errors)

	require.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}
