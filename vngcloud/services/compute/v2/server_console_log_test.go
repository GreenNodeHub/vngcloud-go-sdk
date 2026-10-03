package v2

import (
	lctx "context"
	lhttp "net/http"
	lhttptest "net/http/httptest"
	ltesting "testing"
	ltime "time"

	lsclient "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/client"
	lserr "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/sdk_error"
)

const (
	fakeProjectId = "pro-00000000-0000-0000-0000-000000000000"
	fakeServerId  = "ins-00000000-0000-0000-0000-000000000000"
)

// newFakeComputeService points a ComputeServiceV2 at an httptest server and
// hands it a static token so no IAM call is made.
func newFakeComputeService(pt *ltesting.T, phandler lhttp.HandlerFunc) *ComputeServiceV2 {
	pt.Helper()
	srv := lhttptest.NewServer(phandler)
	pt.Cleanup(srv.Close)

	hc := lsclient.NewHttpClient(lctx.Background()).
		WithRetryCount(0).
		WithReauthFunc(lsclient.IamOauth2, func() (lsclient.ISdkAuthentication, lserr.IError) {
			return new(lsclient.SdkAuthentication).
				WithAccessToken("fake-token").
				WithExpiresAt(ltime.Now().Add(ltime.Hour).UnixNano()), nil
		})

	return &ComputeServiceV2{
		VServerClient: lsclient.NewServiceClient().
			WithEndpoint(srv.URL + "/v2/").
			WithProjectId(fakeProjectId).
			WithClient(hc),
	}
}

func TestGetServerConsoleLogReturnsContent(t *ltesting.T) {
	var gotMethod, gotPath string
	svc := newFakeComputeService(t, func(w lhttp.ResponseWriter, r *lhttp.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"BdsDxe: loading Boot0001\n\nnode-1 login: "}`))
	})

	got, sdkerr := svc.GetServerConsoleLog(NewGetServerConsoleLogRequest(fakeServerId))
	if sdkerr != nil {
		t.Fatalf("expected nil error, got %v", sdkerr)
	}

	if gotMethod != lhttp.MethodGet {
		t.Errorf("method = %s, want GET", gotMethod)
	}

	if want := "/v2/" + fakeProjectId + "/servers/" + fakeServerId + "/console-log"; gotPath != want {
		t.Errorf("path = %s, want %s", gotPath, want)
	}

	if got == nil || got.Content != "BdsDxe: loading Boot0001\n\nnode-1 login: " {
		t.Fatalf("unexpected console log: %+v", got)
	}
}

func TestGetServerConsoleLogServerNotFound(t *ltesting.T) {
	svc := newFakeComputeService(t, func(w lhttp.ResponseWriter, _ *lhttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(lhttp.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Cannot get server with id ` + fakeServerId + `"}`))
	})

	got, sdkerr := svc.GetServerConsoleLog(NewGetServerConsoleLogRequest(fakeServerId))
	if got != nil {
		t.Fatalf("expected nil console log, got %+v", got)
	}

	if sdkerr == nil || !sdkerr.IsError(lserr.EcVServerServerNotFound) {
		t.Fatalf("expected %s, got %v", lserr.EcVServerServerNotFound, sdkerr)
	}
}
