package testhelper

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var (
	// Mux is a multiplexer that can be used to register handlers.
	Mux *http.ServeMux

	// Server is an in-memory HTTP server for testing.
	Server *httptest.Server
)

const genericMultipartBoundary = "--generic-multipart-boundary"

// SetupHTTP prepares the Mux and Server.
func SetupHTTP() {
	Mux = http.NewServeMux()
	Server = httptest.NewServer(Mux)
}

// TeardownHTTP releases HTTP-related resources.
func TeardownHTTP() {
	Server.Close()
}

// Endpoint returns a fake endpoint that will actually target the Mux.
func Endpoint() string {
	return Server.URL + "/"
}

// TestFormValues ensures that all the URL parameters given to the http.Request are the same as values.
func TestFormValues(t *testing.T, r *http.Request, values map[string]string) {
	want := url.Values{}
	for k, v := range values {
		want.Add(k, v)
	}

	r.ParseForm()
	if !reflect.DeepEqual(want, r.Form) {
		t.Errorf("Request parameters = %v, want %v", r.Form, want)
	}
}

// TestMethod checks that the Request has the expected method (e.g. GET, POST).
func TestMethod(t *testing.T, r *http.Request, expected string) {
	if expected != r.Method {
		t.Errorf("Request method = %v, expected %v", r.Method, expected)
	}
}

// TestHeader checks that the header on the http.Request matches the expected value.
func TestHeader(t *testing.T, r *http.Request, header string, expected string) {
	if actual := r.Header.Get(header); expected != actual {
		t.Errorf("Header %s = %s, expected %s", header, actual, expected)
	}
}

// TestBody verifies that the request body matches an expected body.
func TestBody(t *testing.T, r *http.Request, expected string) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Errorf("Unable to read body: %v", err)
	}
	str := string(b)
	if expected != str {
		t.Errorf("Body = %s, expected %s", str, expected)
	}
}

// TestJSONRequest verifies that the JSON payload of a request matches an expected structure, without asserting things about
// whitespace or ordering.
func TestJSONRequest(t *testing.T, r *http.Request, expected string) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Errorf("Unable to read request body: %v", err)
	}

	var actualJSON interface{}
	err = json.Unmarshal(b, &actualJSON)
	if err != nil {
		t.Errorf("Unable to parse request body as JSON: %v", err)
	}

	CheckJSONEquals(t, expected, actualJSON)
}

func normalizeMultipartBoundary(body string) string {
	re := regexp.MustCompile(`^--[a-z0-9]{60}`)
	boundary := re.FindString(body)
	if boundary == "" {
		return ""
	}

	return strings.Replace(body, boundary, genericMultipartBoundary, -1)
}

// TestMultipartRequest verifies that the multipart payload of a request matches an expected structure, without asserting
// errors relating to multipart section boundary markers.  These markers are inserted with random values therefore
// it would not be possible to compare a request payload with an expected value unless there was a way to
// use a hardcoded section boundary.
func TestMultipartRequest(t *testing.T, r *http.Request, expected string) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Errorf("Unable to read request body: %v", err)
	}

	request := normalizeMultipartBoundary(string(b))
	if request == "" {
		t.Errorf("Unable to normalize multipart separator in request body")
	}

	// Requests are formed with \r\n newline patterns therefore remove the
	// carriage return (\r) since it is not possible to embed them within
	// literal strings.
	re := regexp.MustCompile(`\r`)
	request = re.ReplaceAllString(request, "")

	expected = normalizeMultipartBoundary(expected)
	if expected == "" {
		t.Errorf("Unable to normalize multipart separator in expected request")
	}

	if expected != request {
		t.Errorf("Body = xxx%sxxx, expected xxx%sxxx", request, expected)
	}
}