package httpreq

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"k8s.io/klog"
)

// Request allows for building up a request to a server in a chained fashion.
// Any errors are stored until the end of your call, so you only have to
// check once.
type Request struct {
	// required
	client *http.Client
	verb   string

	baseURL *url.URL
	timeout time.Duration

	params  url.Values
	headers http.Header

	// output
	err  error
	body io.Reader

	// The constructed request and the response
	req  *http.Request
	resp *http.Response
}

// NewRequest creates a new request helper object for accessing runtime.Objects on a server.
func NewRequest(client *http.Client, verb string, baseURL *url.URL) *Request {
	r := &Request{
		client:  client,
		verb:    verb,
		baseURL: baseURL,
	}
	r.SetHeader("Accept", "*/*")

	return r
}

//SetParam set parameter for url
func (r *Request) SetParam(paramName, value string) *Request {
	if r.params == nil {
		r.params = make(url.Values)
	}
	r.params[paramName] = append(r.params[paramName], value)
	return r
}

//SetHeader set header for http request
func (r *Request) SetHeader(key, value string) *Request {
	if r.headers == nil {
		r.headers = http.Header{}
	}
	r.headers.Set(key, value)
	return r
}

// URL returns the current working URL.
func (r *Request) URL() *url.URL {

	finalURL := &url.URL{}
	if r.baseURL != nil {
		*finalURL = *r.baseURL
	}

	//if not give params.disable encode query
	//may be we give query line in base url
	if len(r.params) > 0 {
		query := url.Values{}
		for key, values := range r.params {
			for _, value := range values {
				query.Add(key, value)
			}
		}

		finalURL.RawQuery = query.Encode()
	}

	return finalURL
}

// Body makes the request use obj as the body. Optional.
// If obj is a string, try to read a file of that name.
// If obj is a []byte, send it directly.
// If obj is an io.Reader, use it directly.
// If obj is a runtime.Object, marshal it correctly, and set Content-Type header.
// If obj is a runtime.Object and nil, do nothing.
// Otherwise, set an error.
func (r *Request) Body(obj interface{}) *Request {
	if r.err != nil {
		return r
	}
	switch t := obj.(type) {
	case string:
		data, err := ioutil.ReadFile(t)
		if err != nil {
			r.err = err
			return r
		}
		glogBody("Request Body", data)
		r.body = bytes.NewReader(data)
	case []byte:
		glogBody("Request Body", t)
		r.body = bytes.NewReader(t)
	case io.Reader:
		r.body = t
	default:
		r.err = fmt.Errorf("unknown type used for body: %+v", obj)
	}
	return r
}

// glogBody logs a body output that could be either JSON or protobuf. It explicitly guards against
// allocating a new string for the body output unless necessary. Uses a simple heuristic to determine
// whether the body is printable.
func glogBody(prefix string, body []byte) {
	if klog.V(8) {
		if bytes.IndexFunc(body, func(r rune) bool {
			return r < 0x0a
		}) != -1 {
			klog.Infof("%s:\n%s", prefix, hex.Dump(body))
		} else {
			klog.Infof("%s: %s", prefix, string(body))
		}
	}
}

// Request implement send request to remote server and extract response
func (r *Request) Request(fn func(http.Request, *http.Response) error) error {
	//Metrics for total request latency
	start := time.Now()

	if r.err != nil {
		klog.V(4).Infof("Error in request: %v", r.err)
		return r.err
	}

	client := r.client
	if client == nil {
		client = http.DefaultClient
	}

	url := r.URL().String()
	req, err := http.NewRequest(r.verb, url, r.body)
	if err != nil {
		return err
	}
	req.Header = r.headers

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	done := func() bool {
		// Ensure the response body is fully read and closed
		// before we reconnect, so that we reuse the same TCP
		// connection.
		defer func() {
			const maxBodySlurpSize = 2 << 10
			if resp.ContentLength <= maxBodySlurpSize {
				io.Copy(ioutil.Discard, &io.LimitedReader{R: resp.Body, N: maxBodySlurpSize})
			}
			resp.Body.Close()
		}()

		err = fn(*req, resp)
		return true
	}()

	klog.V(9).Infof("request method(%v) (url:%v) end result(%v) Spend time (%vs)",
		r.verb, url, done, time.Now().Second()-start.Second())
	return err
}
