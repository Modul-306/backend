package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTenant(t *testing.T) {
	arg := CreateTenantParams{
		Name: "Test Farm",
		Slug: "test-farm",
	}

	tenant, err := testQueries.CreateTenant(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, tenant)

	require.Equal(t, arg.Name, tenant.Name)
	require.Equal(t, arg.Slug, tenant.Slug)

	require.NotZero(t, tenant.ID)
	require.NotZero(t, tenant.CreatedAt)
}

func TestGetTenantBySlug(t *testing.T) {
	tenant1, err := testQueries.CreateTenant(context.Background(), CreateTenantParams{
		Name: "Get Test",
		Slug: "get-test",
	})
	require.NoError(t, err)

	tenant2, err := testQueries.GetTenantBySlug(context.Background(), tenant1.Slug)
	require.NoError(t, err)
	require.NotEmpty(t, tenant2)

	require.Equal(t, tenant1.ID, tenant2.ID)
	require.Equal(t, tenant1.Name, tenant2.Name)
	require.Equal(t, tenant1.Slug, tenant2.Slug)
}
