package jiracapture

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	searchPath     = "/rest/api/2/search"
	myselfPath     = "/rest/api/2/myself"
	captureTimeout = 5 * time.Minute
	pollInterval   = 1 * time.Second
)

// loginIndicators are URL substrings that mean the browser is still on an
// SSO/login page rather than an authenticated Jira page.
var loginIndicators = []string{"login.jsp", "/login", "okta.com", "sso", "authorize"}

// Capture opens a headed Chrome browser, navigates to the Jira base URL, and
// polls until the user has completed SSO login and landed on an authenticated
// Jira page. At that point it extracts session cookies from the browser,
// writes them to curlFilePath for future reuse, and returns the raw cookie
// string.
//
// Chrome must be installed (standard macOS/Linux install is sufficient — no
// additional driver download is required).
func Capture(baseURL, curlFilePath string) (string, error) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false),
			chromedp.Flag("disable-background-timer-throttling", true),
		)...,
	)
	defer allocCancel()

	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, captureTimeout)
	defer timeoutCancel()

	target := baseURL + "/secure/Dashboard.jspa"
	fmt.Fprintf(os.Stderr, "\n==> Opening browser: %s\n", target)
	fmt.Fprintln(os.Stderr, "    Log in via SSO if prompted, then wait for the Jira dashboard to fully load.")
	fmt.Fprintf(os.Stderr, "    Waiting up to %v...\n\n", captureTimeout)

	// Enable network monitoring and navigate. Navigation returns once the
	// initial page (dashboard or SSO login) fires its load event — which is
	// fast. We handle navigation errors gracefully since a redirect through
	// an SSO provider will surface as an error on some Chrome builds.
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(target)); err != nil {
		if !isContextErr(err) {
			fmt.Fprintf(os.Stderr, "    (navigation: %v — waiting for login)\n", err)
		}
	}

	// Poll the current URL every second. Once the user completes SSO login,
	// Jira redirects back to an authenticated page — at which point the
	// session cookies are present in the browser and ready to capture.
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	attempt := 0
	for {
		select {
		case <-ticker.C:
			attempt++
			cookie, status, err := tryCaptureCookies(ctx, baseURL)
			if err != nil || cookie == "" {
				// Surface periodic status so a stuck capture (e.g. a WAF
				// or proxy rejecting the verification call) is diagnosable
				// instead of silently running out the clock.
				if attempt%15 == 0 {
					fmt.Fprintf(os.Stderr, "    still waiting… last auth check: status=%d err=%v\n", status, err)
				}
				continue
			}
			if err := saveCurlFile(curlFilePath, baseURL, cookie); err != nil {
				fmt.Fprintf(os.Stderr, "warn: could not save curl file %q: %v\n", curlFilePath, err)
			} else {
				fmt.Fprintf(os.Stderr, "==> Session saved to %s\n\n", curlFilePath)
			}
			return cookie, nil

		case <-ctx.Done():
			return "", fmt.Errorf("timed out after %v — did you complete SSO login in the browser?", captureTimeout)
		}
	}
}

// tryCaptureCookies checks whether the browser has navigated away from an
// SSO/login page and, if so, extracts cookies and verifies them against a
// real authenticated Jira endpoint before accepting them. Returns
// ("", status, nil) when the user is not yet authenticated; status is the
// last HTTP status observed from the verification call (0 if none was made
// yet), surfaced purely for diagnostics.
//
// A URL/cookie-name heuristic alone is not reliable: Jira issues a
// JSESSIONID on the very first anonymous page load, before SSO completes, so
// "a session cookie exists" is not proof of being logged in. The only
// trustworthy signal is a successful call to an endpoint that requires auth
// — and that call is made from inside the browser tab (via fetch), not a
// separate Go HTTP client, so it carries the exact headers/TLS fingerprint
// that already got the user past any corporate proxy or WAF.
func tryCaptureCookies(ctx context.Context, baseURL string) (string, int, error) {
	var currentURL string
	if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
		return "", 0, nil // transient — ignore and retry
	}
	if !strings.HasPrefix(currentURL, baseURL) {
		return "", 0, nil
	}
	lower := strings.ToLower(currentURL)
	for _, indicator := range loginIndicators {
		if strings.Contains(lower, indicator) {
			return "", 0, nil
		}
	}

	status, err := checkAuthStatus(ctx, baseURL)
	if err != nil || status != 200 {
		return "", status, err
	}

	var cookies []*network.Cookie
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(c)
		return err
	})); err != nil {
		return "", status, fmt.Errorf("get cookies: %w", err)
	}
	if len(cookies) == 0 {
		return "", status, nil
	}

	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; "), status, nil
}

// checkAuthStatus runs a fetch() inside the active browser tab against an
// endpoint that requires auth, returning the HTTP status observed (0 if the
// fetch itself failed, e.g. network error).
func checkAuthStatus(ctx context.Context, baseURL string) (int, error) {
	js := fmt.Sprintf(
		`fetch(%q, {credentials: 'include', headers: {'Accept': 'application/json'}}).then(r => r.status).catch(() => 0)`,
		baseURL+myselfPath,
	)
	var status int
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &status, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
		return p.WithAwaitPromise(true)
	}))
	return status, err
}

func isContextErr(err error) bool {
	s := err.Error()
	return strings.Contains(s, "context canceled") || strings.Contains(s, "context deadline exceeded")
}

// saveCurlFile writes a curl command in the format parseCurlCookies expects,
// so the next run can skip browser capture entirely.
func saveCurlFile(path, baseURL, cookie string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("curl '%s%s' -b '%s'\n", baseURL, searchPath, cookie)
	return os.WriteFile(path, []byte(content), 0o600)
}
