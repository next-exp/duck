package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetToken_ReturnsToken(t *testing.T) {
	expectedToken := "test-token-12345"

	server := NewDuckAPIServerWithQuerier(nil, "", expectedToken, false)
	req := connect.NewRequest(&pb.GetTokenRequest{})

	resp, err := server.GetToken(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, expectedToken, resp.Msg.Token)
}

func TestGetToken_EmptyToken(t *testing.T) {
	expectedToken := ""

	server := NewDuckAPIServerWithQuerier(nil, "", expectedToken, false)
	req := connect.NewRequest(&pb.GetTokenRequest{})

	resp, err := server.GetToken(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, expectedToken, resp.Msg.Token)
}

func TestGetToken_TokenWithSpecialCharacters(t *testing.T) {
	expectedToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.signed-token.with.dots"

	server := NewDuckAPIServerWithQuerier(nil, "", expectedToken, false)
	req := connect.NewRequest(&pb.GetTokenRequest{})

	resp, err := server.GetToken(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, expectedToken, resp.Msg.Token)
}
