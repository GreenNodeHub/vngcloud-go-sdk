package entity

import (
	lsvcClient "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/client"
)

type AccessToken struct {
	Token     string
	ExpiresAt int64
}

func (s *AccessToken) ToSdkAuthentication() lsvcClient.ISdkAuthentication {
	return new(lsvcClient.SdkAuthentication).
		WithAccessToken(s.Token).
		WithExpiresAt(s.ExpiresAt)
}
