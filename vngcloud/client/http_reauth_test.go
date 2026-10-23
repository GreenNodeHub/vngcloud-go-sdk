package client

import (
	lctx "context"
	lfmt "fmt"
	lhttp "net/http"
	lhttptest "net/http/httptest"
	lstr "strings"
	latomic "sync/atomic"
	ltesting "testing"
	ltime "time"

	lserr "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/sdk_error"
)

// reauthCap bounds how many tokens a test lets the client fetch. The code under test used to
// reauthenticate without limit; without a cap a regression would hang the test instead of failing
// it, so the fake IAM refuses once the cap is reached and the assertions report the count.
const reauthCap = 10

// fakeIam counts token requests and hands out tokens that live for plifetime.
type fakeIam struct {
	calls    latomic.Int32
	lifetime ltime.Duration
}

func (s *fakeIam) reauth() (ISdkAuthentication, lserr.IError) {
	n := s.calls.Add(1)
	if n > reauthCap {
		return nil, lserr.ErrorHandler(lfmt.Errorf("fake IAM: token request #%d is over the cap", n))
	}

	return new(SdkAuthentication).
		WithAccessToken(lfmt.Sprintf("token-%d", n)).
		WithExpiresAt(ltime.Now().Add(s.lifetime).UnixNano()), nil
}

func newReauthClient(piam *fakeIam) *httpClient {
	hc := NewHttpClient(lctx.Background()).(*httpClient)
	hc.WithRetryCount(0)
	hc.WithReauthFunc(IamOauth2, piam.reauth)
	return hc
}

// withValidToken gives the client a token up front, the steady state of a long-running controller.
func withValidToken(phc *httpClient) *httpClient {
	phc.accessToken = new(SdkAuthentication).
		WithAccessToken("token-initial").
		WithExpiresAt(ltime.Now().Add(ltime.Hour).UnixNano())
	phc.WithKvDefaultHeaders("Authorization", "Bearer token-initial")
	return phc
}

func getRequest() IRequest {
	return NewRequest().WithRequestMethod(MethodGet).WithOkCodes(lhttp.StatusOK)
}

// An endpoint that rejects every token used to send DoRequest into reauthenticate -> DoRequest
// recursion with no bottom: one IAM token request per round, a goroutine stack that grew until
// the process was OOM-killed, and a caller that never got an answer. One fresh token is enough to
// tell "stale token" from "this endpoint will not accept us".
func TestPersistent401ReauthenticatesOnceThenFails(t *ltesting.T) {
	var hits latomic.Int32
	srv := lhttptest.NewServer(lhttp.HandlerFunc(func(w lhttp.ResponseWriter, _ *lhttp.Request) {
		hits.Add(1)
		w.WriteHeader(lhttp.StatusUnauthorized)
	}))
	defer srv.Close()

	iam := &fakeIam{lifetime: ltime.Hour}
	_, sdkErr := withValidToken(newReauthClient(iam)).DoRequest(srv.URL, getRequest())

	if got := iam.calls.Load(); got != 1 {
		t.Fatalf("token requests = %d, want exactly 1", got)
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("requests to the endpoint = %d, want 2 (stale token, then fresh token)", got)
	}
	assertUnauthorized(t, sdkErr)
}

// When the token is fetched at the start of the call, that fetch is the one reauthentication the
// call gets: a 401 on a token that is seconds old is not a stale token.
func TestPersistent401WithFreshlyFetchedTokenDoesNotFetchAgain(t *ltesting.T) {
	var hits latomic.Int32
	srv := lhttptest.NewServer(lhttp.HandlerFunc(func(w lhttp.ResponseWriter, _ *lhttp.Request) {
		hits.Add(1)
		w.WriteHeader(lhttp.StatusUnauthorized)
	}))
	defer srv.Close()

	iam := &fakeIam{lifetime: ltime.Hour}
	_, sdkErr := newReauthClient(iam).DoRequest(srv.URL, getRequest())

	if got := iam.calls.Load(); got != 1 {
		t.Fatalf("token requests = %d, want exactly 1", got)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("requests to the endpoint = %d, want 1", got)
	}
	assertUnauthorized(t, sdkErr)
}

// The case the reauth path exists for must keep working: a token that went stale is replaced and
// the request goes through.
func TestSingle401IsRecoveredByReauthentication(t *ltesting.T) {
	var hits latomic.Int32
	srv := lhttptest.NewServer(lhttp.HandlerFunc(func(w lhttp.ResponseWriter, r *lhttp.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") == "Bearer token-initial" {
			w.WriteHeader(lhttp.StatusUnauthorized)
			return
		}
		w.WriteHeader(lhttp.StatusOK)
	}))
	defer srv.Close()

	iam := &fakeIam{lifetime: ltime.Hour}
	if _, sdkErr := withValidToken(newReauthClient(iam)).DoRequest(srv.URL, getRequest()); sdkErr != nil {
		t.Fatalf("request failed after reauthentication: %v", sdkErr.GetError())
	}
	if got := iam.calls.Load(); got != 1 {
		t.Fatalf("token requests = %d, want 1", got)
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("requests to the endpoint = %d, want 2", got)
	}
}

// NeedReauth reports true for any token with less than five minutes left. If IAM issues tokens
// shorter than that, the token just fetched already "needs reauth", and the pre-request path used
// to recurse on it forever without ever sending the request.
func TestShortLivedTokenDoesNotLoopBeforeSending(t *ltesting.T) {
	var hits latomic.Int32
	srv := lhttptest.NewServer(lhttp.HandlerFunc(func(w lhttp.ResponseWriter, _ *lhttp.Request) {
		hits.Add(1)
		w.WriteHeader(lhttp.StatusOK)
	}))
	defer srv.Close()

	iam := &fakeIam{lifetime: ltime.Minute}
	if _, sdkErr := newReauthClient(iam).DoRequest(srv.URL, getRequest()); sdkErr != nil {
		t.Fatalf("request failed: %v", sdkErr.GetError())
	}
	if got := iam.calls.Load(); got != 1 {
		t.Fatalf("token requests = %d, want 1", got)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("requests to the endpoint = %d, want 1", got)
	}
}

// The incident behind this test: the GLB endpoint answered 301 to another host, Go dropped the
// Authorization header on the cross-host hop, and the target answered 401. No token can fix that,
// so the client must stop after one reauthentication.
func TestCrossHostRedirectThatDropsAuthorizationIsBounded(t *ltesting.T) {
	var targetHits latomic.Int32
	target := lhttptest.NewServer(lhttp.HandlerFunc(func(w lhttp.ResponseWriter, r *lhttp.Request) {
		targetHits.Add(1)
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(lhttp.StatusUnauthorized)
			return
		}
		w.WriteHeader(lhttp.StatusOK)
	}))
	defer target.Close()

	// The origin is reached as 127.0.0.1 and redirects to localhost: a different host name, so Go
	// will not forward the Authorization header.
	targetURL := lstr.Replace(target.URL, "127.0.0.1", "localhost", 1)
	origin := lhttptest.NewServer(lhttp.HandlerFunc(func(w lhttp.ResponseWriter, r *lhttp.Request) {
		lhttp.Redirect(w, r, targetURL+r.URL.RequestURI(), lhttp.StatusMovedPermanently)
	}))
	defer origin.Close()

	iam := &fakeIam{lifetime: ltime.Hour}
	_, sdkErr := withValidToken(newReauthClient(iam)).DoRequest(origin.URL+"/v1/items", getRequest())

	if got := iam.calls.Load(); got != 1 {
		t.Fatalf("token requests = %d, want exactly 1", got)
	}
	if got := targetHits.Load(); got != 2 {
		t.Fatalf("requests reaching the redirect target = %d, want 2", got)
	}
	assertUnauthorized(t, sdkErr)
}

func assertUnauthorized(t *ltesting.T, psdkErr lserr.IError) {
	t.Helper()

	if psdkErr == nil {
		t.Fatal("persistent 401 produced no error")
	}
	if got := psdkErr.GetErrorCode(); got != lserr.EcUnauthorized {
		t.Fatalf("error code = %q, want %q (error: %v)", got, lserr.EcUnauthorized, psdkErr.GetError())
	}
	// Consumers commonly do `return sdkErr.GetError()`; a nil here reads as success.
	if psdkErr.GetError() == nil {
		t.Fatal("GetError() is nil, a caller returning it would report success")
	}
	if code, _ := psdkErr.GetParameters()["statusCode"].(int); code != lhttp.StatusUnauthorized {
		t.Fatalf("statusCode parameter = %v, want 401", psdkErr.GetParameters()["statusCode"])
	}
}
