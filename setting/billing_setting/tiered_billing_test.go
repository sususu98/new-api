package billing_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/stretchr/testify/require"
)

func TestGPT55DefaultBillingExpr(t *testing.T) {
	expr, ok := GetBillingExpr("gpt-5.5")
	require.True(t, ok)
	require.Equal(t, BillingModeTieredExpr, GetBillingMode("gpt-5.5"))
	require.Equal(t, BillingModeTieredExpr, GetBillingMode("gpt-5.5-2026-04-23"))
	require.Equal(t, BillingModeRatio, GetBillingMode("gpt-5.5-fast"))
	require.NoError(t, SmokeTestExpr(expr))

	params := billingexpr.TokenParams{P: 1000, C: 100, CR: 50, Len: 1000}

	standard, trace, err := billingexpr.RunExprWithRequest(expr, params, billingexpr.RequestInput{})
	require.NoError(t, err)
	require.Equal(t, "standard", trace.MatchedTier)
	require.InDelta(t, 8025, standard, 0.001)

	priority, trace, err := billingexpr.RunExprWithRequest(expr, params, billingexpr.RequestInput{
		Body: []byte(`{"_newapi_actual_service_tier":"priority"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "priority", trace.MatchedTier)
	require.InDelta(t, 16050, priority, 0.001)

	priorityLong, trace, err := billingexpr.RunExprWithRequest(expr, billingexpr.TokenParams{P: 1000, C: 100, CR: 50, Len: 300000}, billingexpr.RequestInput{
		Body: []byte(`{"_newapi_actual_service_tier":"priority"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "priority_long_context", trace.MatchedTier)
	require.InDelta(t, 29100, priorityLong, 0.001)

	requestedPriority, trace, err := billingexpr.RunExprWithRequest(expr, params, billingexpr.RequestInput{
		Body: []byte(`{"service_tier":"priority"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "standard", trace.MatchedTier)
	require.InDelta(t, standard, requestedPriority, 0.001)

	auto, trace, err := billingexpr.RunExprWithRequest(expr, params, billingexpr.RequestInput{
		Body: []byte(`{"_newapi_actual_service_tier":"auto"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "standard", trace.MatchedTier)
	require.InDelta(t, standard, auto, 0.001)

	flexLong, trace, err := billingexpr.RunExprWithRequest(expr, billingexpr.TokenParams{P: 1000, C: 100, CR: 50, Len: 300000}, billingexpr.RequestInput{
		Body: []byte(`{"_newapi_actual_service_tier":"flex"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "flex_long_context", trace.MatchedTier)
	require.InDelta(t, 7275, flexLong, 0.001)
}
