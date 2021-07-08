package client

import (
    "testing"
)

// TestCreateChannelNull calls CreateChannel with null host checking
// for an invalid return value.
func TestCreateChannelNull(t *testing.T) {
    client, err := CreateChannel("", 0)
    if client != nil || err == nil {
        t.Fatalf(`CreateChannel("", 0) = nil, err `)
    }
}

// TestCreateChannelValid calls CreateChannel with valid host, checking
// for an valid return value.
/*func TestCreateChannelValid(t *testing.T) {
    client, err := CreateChannel("", 0)
    if client != nil || err == nil {
        t.Fatalf(`CreateChannel("", 0) = nil, err `)
    }
}*/
