package httpclient

import "net/http"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

//type DefaultHTTPClient struct {
//	Client *http.Client
//}
//
//func NewDefaultHTTPClient() *DefaultHTTPClient {
//	return &DefaultHTTPClient{
//		Client: &http.Client{},
//	}
//}
//
//func (c *DefaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
//	return c.Client.Do(req)
//}
