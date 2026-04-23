package main

import (
	"database/sql"

	"github.com/jmbenlloch/next_duck/pkg/database"
)

// DuckAPIServer implements DuckAPIHandler interface
type DuckAPIServer struct {
	queries          database.Querier
	rpcClient        RPCClient
	configFilename   string
	centrifugalToken string
	devVersion       bool
	runTransition    RunTransition
}

// NewDuckAPIServer creates a new DuckAPIServer instance
func NewDuckAPIServer(db *sql.DB, configFilename string, centrifugalToken string, devVersion bool) *DuckAPIServer {
	queries := database.New(db)
	return &DuckAPIServer{
		queries:          queries,
		rpcClient:        NewRealRPCClient(),
		configFilename:   configFilename,
		centrifugalToken: centrifugalToken,
		devVersion:       devVersion,
	}
}

// NewDuckAPIServerWithQuerier creates a new DuckAPIServer with a custom Querier (for testing)
func NewDuckAPIServerWithQuerier(queries database.Querier, configFilename string, centrifugalToken string, devVersion bool) *DuckAPIServer {
	return &DuckAPIServer{
		queries:          queries,
		rpcClient:        nil, // Can be set separately for testing
		configFilename:   configFilename,
		centrifugalToken: centrifugalToken,
		devVersion:       devVersion,
	}
}

// NewDuckAPIServerWithMocks creates a new DuckAPIServer with custom Querier and RPCClient (for testing)
func NewDuckAPIServerWithMocks(queries database.Querier, rpcClient RPCClient, configFilename string, centrifugalToken string, devVersion bool) *DuckAPIServer {
	return &DuckAPIServer{
		queries:          queries,
		rpcClient:        rpcClient,
		configFilename:   configFilename,
		centrifugalToken: centrifugalToken,
		devVersion:       devVersion,
	}
}
