package auth_test

import (
	"testing"

	"github.com/jj-style/eventpix/internal/pkg/utils/auth"
	"github.com/stretchr/testify/require"
)

func TestEncryptAndHashComparePassword(t *testing.T) {
	t.Parallel()

	got, err := auth.EncryptPassword("password")
	require.NoError(t, err)
	require.NotEqual(t, "password", got)

	require.True(t, auth.ComparePassword("password", got))
	require.False(t, auth.ComparePassword("wrong password", got))
}

func TestToken(t *testing.T) {
	t.Parallel()

	secret := "secret key"

	t.Run("happy", func(t *testing.T) {
		t.Parallel()

		got, err := auth.CreateToken(secret, "username")
		require.NoError(t, err)

		claims, err := auth.VerifyToken(secret, got)
		require.NoError(t, err)
		subject, err := claims.GetSubject()
		require.NoError(t, err)
		require.Equal(t, "username", subject)
	})

	t.Run("unhappy", func(t *testing.T) {
		t.Parallel()

		got, err := auth.CreateToken(secret, "username")
		require.NoError(t, err)

		claims, err := auth.VerifyToken("wrong secret", got)
		require.Error(t, err)
		require.Nil(t, claims)
	})
}
