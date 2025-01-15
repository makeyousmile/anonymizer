package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// HTTP-клиент с настройкой прокси
var httpClient = createProxyClient("http://8xU0Rarogj:FvbnaaZBZp@v.curly.team:37848")

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		targetURL := r.URL.Query().Get("url")
		if targetURL == "" {
			fmt.Fprintln(w, "Usage: /?url=http://example.com")
			return
		}

		// Fetch the target content
		resp, err := httpClient.Get(targetURL)
		if err != nil {
			http.Error(w, "Failed to fetch the URL", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copy headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Rewrite links and resources in the body if the content is HTML
		if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "Failed to read the response body", http.StatusInternalServerError)
				return
			}
			updatedBody := rewriteHTML(string(body), r.Host)
			w.WriteHeader(resp.StatusCode)
			w.Write([]byte(updatedBody))
		} else {
			// Non-HTML content, just pipe it through
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
		}
	})

	fmt.Println("Starting proxy on :8080")
	http.ListenAndServe(":8080", nil)
}

// createProxyClient создает HTTP-клиент с прокси-сервером
func createProxyClient(proxyURL string) *http.Client {
	parsedProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		panic(fmt.Sprintf("Invalid proxy URL: %s", proxyURL))
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(parsedProxyURL),
	}
	fmt.Sprintf("Invalid proxy URL: %s", transport)
	return &http.Client{
		//Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// rewriteHTML modifies all URLs in HTML content to point through the anonymizer
func rewriteHTML(htmlContent, proxyHost string) string {
	// Rewrite href attributes
	htmlContent = rewriteAttributes(htmlContent, "href", proxyHost)

	// Rewrite src attributes (images, scripts, etc.)
	htmlContent = rewriteAttributes(htmlContent, "src", proxyHost)

	// Inject a JavaScript snippet to rewrite dynamically generated URLs
	htmlContent = injectJavaScript(htmlContent, proxyHost)

	return htmlContent
}

// rewriteAttributes replaces attribute values with anonymized URLs
func rewriteAttributes(content, attribute, proxyHost string) string {
	attributeRegex := regexp.MustCompile(fmt.Sprintf(`%s=["']([^"']+)["']`, attribute))
	return attributeRegex.ReplaceAllStringFunc(content, func(attr string) string {
		matches := attributeRegex.FindStringSubmatch(attr)
		if len(matches) < 2 {
			return attr
		}
		originalURL := matches[1]

		// Skip absolute URLs that are not HTTP(S)
		if strings.HasPrefix(originalURL, "data:") || strings.HasPrefix(originalURL, "javascript:") {
			return attr
		}

		// Generate the proxy URL
		proxyURL := fmt.Sprintf("http://%s/?url=%s", proxyHost, url.QueryEscape(originalURL))
		return strings.Replace(attr, originalURL, proxyURL, 1)
	})
}

// injectJavaScript adds a script to rewrite dynamically generated URLs
func injectJavaScript(htmlContent, proxyHost string) string {
	jsSnippet := fmt.Sprintf(`
<script>
document.addEventListener("DOMContentLoaded", function() {
    function rewriteDynamicURLs(attr) {
        document.querySelectorAll("[" + attr + "]").forEach(function(element) {
            var original = element.getAttribute(attr);
            if (original && !original.startsWith("data:") && !original.startsWith("javascript:")) {
                var proxyURL = "http://%s/?url=" + encodeURIComponent(original);
                element.setAttribute(attr, proxyURL);
            }
        });
    }
    rewriteDynamicURLs("href");
    rewriteDynamicURLs("src");
});
</script>
`, proxyHost)

	// Inject JavaScript before closing </body> tag
	bodyCloseTag := "</body>"
	if strings.Contains(htmlContent, bodyCloseTag) {
		return strings.Replace(htmlContent, bodyCloseTag, jsSnippet+bodyCloseTag, 1)
	}
	return htmlContent + jsSnippet
}
