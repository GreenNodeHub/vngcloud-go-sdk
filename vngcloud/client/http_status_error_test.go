package client

import (
	lctx "context"
	lhttp "net/http"
	lhttptest "net/http/httptest"
	lstr "strings"
	ltesting "testing"

	lserr "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/sdk_error"
)

// doAgainstStatus runs a real GET through DoRequest against a server that answers with pstatus,
// which is the only way to cover the status switch where the mapping lives.
func doAgainstStatus(t *ltesting.T, pstatus int) lserr.IError {
	t.Helper()

	srv := lhttptest.NewServer(lhttp.HandlerFunc(func(w lhttp.ResponseWriter, _ *lhttp.Request) {
		w.WriteHeader(pstatus)
	}))
	defer srv.Close()

	_, sdkErr := NewHttpClient(lctx.Background()).DoRequest(srv.URL,
		NewRequest().WithRequestMethod(MethodGet).WithSkipAuth(true).WithOkCodes(lhttp.StatusOK))
	if sdkErr == nil {
		t.Fatalf("status %d produced no error", pstatus)
	}
	return sdkErr
}

// #33485: 429 and 403 were both reported as EcPermissionDenied, so a consumer could not tell a
// throttle from a real authorisation failure without reading the raw status code out of the
// parameters. Reading it as an authorisation failure is what made a controller back off for 20
// minutes believing it had lost its permissions, while the throttle it was actually hitting kept
// renewing itself.
func TestTooManyRequestsIsNotReportedAsPermissionDenied(t *ltesting.T) {
	sdkErr := doAgainstStatus(t, lhttp.StatusTooManyRequests)

	if got := sdkErr.GetErrorCode(); got != lserr.EcTooManyRequests {
		t.Fatalf("429 error code = %q, want %q", got, lserr.EcTooManyRequests)
	}
	if msg := sdkErr.GetError().Error(); lstr.Contains(lstr.ToLower(msg), "permission denied") {
		t.Fatalf("429 error text still blames permissions: %q", msg)
	}
}

// 403 is the case the old mapping was written for and it must keep saying so.
func TestForbiddenIsStillPermissionDenied(t *ltesting.T) {
	sdkErr := doAgainstStatus(t, lhttp.StatusForbidden)

	if got := sdkErr.GetErrorCode(); got != lserr.EcPermissionDenied {
		t.Fatalf("403 error code = %q, want %q", got, lserr.EcPermissionDenied)
	}
}

// Consumers read the throttle's status code and headers straight out of the parameters -
// lb-controller parses X-Ratelimit-Reset from there to decide how long to requeue. Giving 429 its
// own error code must not take that away.
func TestTooManyRequestsKeepsTheParametersConsumersParse(t *ltesting.T) {
	sdkErr := doAgainstStatus(t, lhttp.StatusTooManyRequests)

	params := sdkErr.GetParameters()
	if code, _ := params["statusCode"].(int); code != lhttp.StatusTooManyRequests {
		t.Fatalf("statusCode parameter = %v, want %d", params["statusCode"], lhttp.StatusTooManyRequests)
	}
	if _, ok := params["responseHeaders"].(lhttp.Header); !ok {
		t.Fatalf("responseHeaders parameter missing or wrong type: %T", params["responseHeaders"])
	}
	if method, _ := params["method"].(string); method != lhttp.MethodGet {
		t.Fatalf("method parameter = %v, want GET", params["method"])
	}
}

// vks-controlplane-logs-worker decides "the cluster is gone" by matching the error text, and only
// skips that check for EcPermissionDenied. Once 429 no longer carries that code it falls through
// to the text, so the text must not read like a missing resource.
func TestTooManyRequestsTextCannotBeReadAsMissing(t *ltesting.T) {
	sdkErr := doAgainstStatus(t, lhttp.StatusTooManyRequests)

	text := lstr.ToLower(sdkErr.GetMessage() + " " + sdkErr.GetError().Error())
	for _, bad := range []string{"not found", "404", "does not exist"} {
		if lstr.Contains(text, bad) {
			t.Fatalf("429 text contains %q, which a consumer reads as a missing resource: %q", bad, text)
		}
	}
}
