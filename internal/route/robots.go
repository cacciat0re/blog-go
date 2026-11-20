package route

import "net/http"

// robotsTXT allows every crawler to read the whole site.
const robotsTXT = `User-agent: *
Allow: /
`

// robots serves the site-wide crawl policy at the root path crawlers request.
func robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write([]byte(robotsTXT))
}
