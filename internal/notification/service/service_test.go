package service

import (
	"testing"

	"github.com/example/adnova/internal/notification/domain"
	"github.com/stretchr/testify/require"
)

func TestNotificationStatusesRemainStable(t *testing.T) {
	require.Equal(t, "UNREAD", domain.StatusUnread)
	require.Equal(t, "READ", domain.StatusRead)
}
