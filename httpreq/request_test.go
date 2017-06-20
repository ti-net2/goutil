package httpreq

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestHTTPRequest(t *testing.T) {
	reqURL, err := url.Parse(fmt.Sprintf("https://www.baidu.com/"))
	if err != nil {
		t.Errorf("parse url failed. error(%v)\r\n", err)
	}
	r := NewRequest(nil, http.MethodPost, reqURL)
	reqerr := r.Request(func(req http.Request, resp *http.Response) error {
		if resp.StatusCode >= 300 {
			t.Errorf("Except 2xx code. but got %d(%s) code", resp.StatusCode, resp.Status)
		}
		return nil
	})
	if reqerr != nil {
		t.Errorf("http request error(%v) ", reqerr)
	}
}
