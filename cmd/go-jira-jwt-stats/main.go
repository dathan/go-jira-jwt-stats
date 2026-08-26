package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/dathan/go-jira-jwt-stats/pkg/dashboard"
	"github.com/dathan/go-jira-jwt-stats/pkg/jiracapture"
)

const (
	defaultBaseURL   = "https://jirasw.nvidia.com"
	searchPath       = "/rest/api/2/search"
	defaultOutput    = "dashboard.html"
	defaultOutputDir = "output"
	pageSize         = 100
)

// ── Jira REST API types ──────────────────────────────────────────────────────

type searchRequest struct {
	JQL        string   `json:"jql"`
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Fields     []string `json:"fields"`
}

type searchResponse struct {
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
	Total      int         `json:"total"`
	Issues     []jiraIssue `json:"issues"`
}

type jiraPerson struct {
	DisplayName string `json:"displayName"`
	Name        string `json:"name"`
}

func (p *jiraPerson) label() string {
	if p == nil {
		return "Unassigned"
	}
	if p.DisplayName != "" {
		return p.DisplayName
	}
	return p.Name
}

type jiraIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
		Priority *struct {
			Name string `json:"name"`
		} `json:"priority"`
		Assignee *jiraPerson `json:"assignee"`
		Reporter *jiraPerson `json:"reporter"`
		Project  struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"project"`
		DueDate *string `json:"duedate"`
		Created string  `json:"created"`
	} `json:"fields"`
}

// ── JS-serializable types (match the dashboard JS object shapes) ──────────────

type jsAssigneeGroup struct {
	Name     string   `json:"name"`
	Count    int      `json:"count"`
	Overdue  int      `json:"overdue"`
	Projects []string `json:"projects"`
}

type jsIssue struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Project  string `json:"project"`
	Scope    string `json:"scope"` // "home" | "external"
	Assignee string `json:"assignee"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Due      string `json:"due"`
	Overdue  bool   `json:"overdue"`
	Created  string `json:"created"`
	AgeDays  int    `json:"ageDays"`
}

type jsCallout struct {
	Name        string   `json:"name"`
	Count       int      `json:"count"`
	TeamAvg     float64  `json:"teamAvg"`
	PctAboveAvg int      `json:"pctAboveAvg"`
	Projects    []string `json:"projects"`
}

// ── Template data ─────────────────────────────────────────────────────────────

type dashData struct {
	FetchedAt         string
	BaseURL           string
	BaseURLJSON       string
	JQLEncoded        string
	TeamLabel         string
	HomeProjects      string
	TotalOpen         int
	HomeCount         int
	ExternalCount     int
	OverdueCount      int
	UnassignedCount   int
	ProjectCount      int
	OldestDays        int
	AvgAgeDays        int
	HighPriorityCount int
	OverloadedCount   int
	AssigneesJSON     string
	ProjectsJSON      string
	StatusJSON        string
	PriorityJSON      string
	OldestJSON        string
	CalloutsJSON      string
	IssuesJSON        string

	// Per-KPI JQL links — each card on the dashboard opens the matching
	// filtered view directly in Jira.
	HomeJQLEncoded         string
	ExternalJQLEncoded     string
	OverdueJQLEncoded      string
	UnassignedJQLEncoded   string
	HighPriorityJQLEncoded string
	AgeJQLEncoded          string
}

// curlCookieRE extracts the -b '...' cookie value from a curl command.
var curlCookieRE = regexp.MustCompile(`(?s)-b\s+'([^']+)'`)

// orderByRE splits a trailing "ORDER BY ..." clause off a JQL string so
// additional AND conditions can be inserted before it.
var orderByRE = regexp.MustCompile(`(?i)\s+order\s+by\s+.*$`)

// splitJQL separates the filter portion of a JQL string from its trailing
// ORDER BY clause (if any).
func splitJQL(jql string) (where, orderBy string) {
	loc := orderByRE.FindStringIndex(jql)
	if loc == nil {
		return jql, ""
	}
	return jql[:loc[0]], jql[loc[0]:]
}

// quoteJQLList renders a Go string slice as a double-quoted, comma-separated
// JQL value list, e.g. ["A","B"] -> `"A", "B"`.
func quoteJQLList(items []string) string {
	quoted := make([]string, len(items))
	for i, v := range items {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(quoted, ", ")
}

// parseCurlCookies reads a curl command string and returns the cookie header value
// found after the -b '...' flag. This lets you paste a curl from DevTools directly
// into a file and avoid shell quoting issues with semicolons in the cookie string.
func parseCurlCookies(curlStr string) (string, error) {
	m := curlCookieRE.FindStringSubmatch(curlStr)
	if m == nil {
		return "", fmt.Errorf("no -b '...' cookie argument found in curl command")
	}
	return m[1], nil
}

// cookiesFromCurlFile reads a curl command from the given path (or stdin if "-")
// and parses the cookie string from it.
func cookiesFromCurlFile(path string) (string, error) {
	var raw []byte
	var err error
	if path == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("read curl file: %w", err)
	}
	return parseCurlCookies(string(raw))
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func main() {
	curlFile := flag.String("curl-file", os.Getenv("JIRA_CURL_FILE"),
		"Path to a file containing the curl command copied from DevTools (or '-' for stdin).\n"+
			"The program will extract the -b '...' cookie from it automatically.\n"+
			"Get it: DevTools → Network → any request → right-click → Copy as cURL → paste into a file.")
	cookies := flag.String("cookies", os.Getenv("JIRA_COOKIES"),
		"Raw Jira cookie string (fallback if --curl-file is not set).\n"+
			"Must be single-quoted to avoid shell issues with semicolons:\n"+
			"  export JIRA_COOKIES='JSESSIONID=...; atlassian.xsrf.token=...'")
	baseURL := flag.String("base-url", envOr("JIRA_BASE_URL", defaultBaseURL), "Jira base URL")
	teamMembers := flag.String("team-members", os.Getenv("JIRA_TEAM_MEMBERS"),
		"Comma-separated Jira usernames whose obligations you want to track (required)")
	projects := flag.String("projects", os.Getenv("JIRA_PROJECTS"),
		"Comma-separated Jira project keys your team owns (e.g. OPPEPROJ). Issues in these\n"+
			"projects are 'home' backlog; issues elsewhere assigned to your team members are\n"+
			"'external' asks. (required)")
	jqlOverride := flag.String("jql", os.Getenv("JIRA_JQL"),
		"Full JQL override. When unset, a JQL is built from --team-members:\n"+
			"  assignee in (...) AND resolution = Unresolved ORDER BY due ASC, created ASC")
	extraJQL := flag.String("jql-extra", os.Getenv("JIRA_JQL_EXTRA"),
		"Extra JQL clause ANDed onto the generated query (ignored if --jql is set)")
	output := flag.String("output", defaultOutput, "Output HTML file basename")
	outputDir := flag.String("output-dir", defaultOutputDir, "Directory to write the dashboard HTML file into")
	open := flag.Bool("open", false, "Open dashboard in browser after writing")
	flag.Parse()

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		log.Fatalf("create output dir %q: %v", *outputDir, err)
	}

	members := splitCSV(*teamMembers)
	homeProjects := splitCSV(*projects)

	var missing []string
	if len(members) == 0 && *jqlOverride == "" {
		missing = append(missing, "--team-members")
	}
	if len(homeProjects) == 0 {
		missing = append(missing, "--projects")
	}
	if len(missing) > 0 {
		usageString(missing)
		os.Exit(1)
	}

	jql := *jqlOverride
	if jql == "" {
		jql = buildJQL(members, *extraJQL)
	}

	cookieStr := *cookies

	// 1. Prefer explicit curl file.
	if *curlFile != "" {
		parsed, err := cookiesFromCurlFile(*curlFile)
		if err != nil {
			log.Printf("auth: curl-file %q unusable (%v) — falling back to browser capture", *curlFile, err)
		} else {
			cookieStr = parsed
			log.Printf("auth: loaded cookies from curl file %q", *curlFile)
		}
	}

	// 2. Fall back to browser capture when no cookies are available.
	if cookieStr == "" {
		curlPath := *curlFile
		if curlPath == "" {
			curlPath = "conf/jira.curl"
		}
		log.Printf("auth: no cookies found — launching browser capture")
		captured, err := jiracapture.Capture(*baseURL, curlPath)
		if err != nil {
			log.Fatalf("browser capture: %v", err)
		}
		cookieStr = captured
	}

	log.Printf("jql: %s", jql)

	issues, err := fetchAll(cookieStr, *baseURL, jql)
	if err != nil {
		if !strings.Contains(err.Error(), "401") {
			log.Fatalf("fetch: %v", err)
		}
		log.Printf("auth: session expired (HTTP 401) — relaunching browser to refresh cookies")
		curlPath := *curlFile
		if curlPath == "" {
			curlPath = "conf/jira.curl"
		}
		refreshed, captureErr := jiracapture.Capture(*baseURL, curlPath)
		if captureErr != nil {
			log.Fatalf("browser capture: %v", captureErr)
		}
		cookieStr = refreshed
		issues, err = fetchAll(cookieStr, *baseURL, jql)
		if err != nil {
			log.Fatalf("fetch after re-auth: %v", err)
		}
	}
	log.Printf("fetched %d open issues", len(issues))

	tmpl, err := template.New("dash").Parse(dashboard.DashboardTemplate)
	if err != nil {
		log.Fatalf("parse template: %v", err)
	}

	data := buildDash(issues, *baseURL, jql, members, homeProjects)

	outPath := filepath.Join(*outputDir, strings.TrimSuffix(*output, ".html")+".html")
	f, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("create %s: %v", outPath, err)
	}
	if err := tmpl.Execute(f, data); err != nil {
		f.Close()
		log.Fatalf("render %s: %v", outPath, err)
	}
	f.Close()

	abs, _ := filepath.Abs(outPath)
	log.Printf("written: %s  (%d issues)", abs, len(issues))

	if *open {
		openBrowser(abs)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// jiraTimeLayouts are the timestamp formats Jira Server/DC commonly returns
// for date-time fields such as "created".
var jiraTimeLayouts = []string{
	"2006-01-02T15:04:05.000-0700",
	time.RFC3339,
}

func parseJiraTime(s string) (time.Time, bool) {
	for _, layout := range jiraTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// overloadMultiplier and minOverloadCount gate the "workload callout" list:
// an assignee is flagged only when they're carrying meaningfully more than
// the team's average open load, not just marginally more.
const (
	overloadMultiplier = 1.5
	minOverloadCount   = 3
	oldestItemsLimit   = 10
)

func isHighPriority(name string) bool {
	l := strings.ToLower(name)
	return strings.Contains(l, "highest") || strings.Contains(l, "high")
}

// buildJQL constructs the default team-obligations JQL: every unresolved
// issue assigned to any of the given team members, across all projects the
// account can see — this is what surfaces asks coming in from outside the
// team's own project(s).
func buildJQL(members []string, extra string) string {
	quoted := make([]string, len(members))
	for i, m := range members {
		quoted[i] = m
	}
	jql := fmt.Sprintf("assignee in (%s) AND resolution = Unresolved", strings.Join(quoted, ", "))
	if extra != "" {
		jql += " AND (" + extra + ")"
	}
	jql += " ORDER BY due ASC, created ASC"
	return jql
}

func usageString(missing []string) {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Missing required flags:")
	for _, m := range missing {
		fmt.Fprintf(os.Stderr, "  • %s\n", m)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Auth — easiest method (paste a curl from DevTools):")
	fmt.Fprintln(os.Stderr, "  1. Go to your Jira site in Chrome and log in via SSO")
	fmt.Fprintln(os.Stderr, "  2. Open DevTools (F12) → Network tab")
	fmt.Fprintln(os.Stderr, "  3. Click any XHR/Fetch request → right-click → Copy as cURL")
	fmt.Fprintln(os.Stderr, "  4. pbpaste > /tmp/jira.curl")
	fmt.Fprintln(os.Stderr, "  5. Re-run with:  --curl-file /tmp/jira.curl --team-members u1,u2 --projects OPPEPROJ")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Or export JIRA_CURL_FILE=/tmp/jira.curl and omit --curl-file.")
	fmt.Fprintln(os.Stderr, "  If no cookies are available at all, a browser window opens for SSO login.")
}

// fetchAll calls the Jira REST search API using browser session cookies and
// paginates via startAt until all results are retrieved.
func fetchAll(cookieStr, baseURL, jql string) ([]jiraIssue, error) {
	apiURL := baseURL + searchPath
	fields := []string{"summary", "status", "priority", "assignee", "reporter", "project", "duedate", "created"}
	var all []jiraIssue
	startAt := 0

	for {
		req := searchRequest{
			JQL:        jql,
			StartAt:    startAt,
			MaxResults: pageSize,
			Fields:     fields,
		}

		body, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}

		httpReq, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(string(body)))
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")
		httpReq.Header.Set("Cookie", cookieStr)
		httpReq.Header.Set("X-Atlassian-Token", "no-check")
		httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36")
		httpReq.Header.Set("Origin", baseURL)
		httpReq.Header.Set("Referer", baseURL+"/issues/")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("http: %w", err)
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			snippet := strings.TrimSpace(string(b))
			if len(snippet) > 500 {
				snippet = snippet[:500] + "…"
			}
			return nil, fmt.Errorf(
				"HTTP %d Unauthorized — session cookies have expired (or were rejected).\n"+
					"  response body: %s\n"+
					"  1. Open your Jira site in Chrome and log in\n"+
					"  2. DevTools → Network → any request → right-click → Copy as cURL\n"+
					"  3. Paste into a file:  pbpaste > /tmp/jira.curl\n"+
					"  4. Re-run:  go run . --curl-file /tmp/jira.curl --open", resp.StatusCode, snippet,
			)
		}
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("Jira returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
		}

		var result searchResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode: %w", err)
		}
		resp.Body.Close()

		all = append(all, result.Issues...)
		startAt += len(result.Issues)
		if len(result.Issues) == 0 || startAt >= result.Total {
			break
		}
	}
	return all, nil
}

// buildDash processes raw issues into the struct the HTML template needs.
func buildDash(issues []jiraIssue, baseURL, jql string, members, homeProjects []string) dashData {
	home := map[string]bool{}
	for _, p := range homeProjects {
		home[strings.ToUpper(p)] = true
	}

	now := time.Now().UTC()
	today := now.Format("2006-01-02")

	type agState struct {
		jsAssigneeGroup
		projectSet map[string]bool
	}
	agMap := map[string]*agState{}
	projectMap := map[string]int{}
	statusMap := map[string]int{}
	priorityMap := map[string]int{}
	jsIssues := make([]jsIssue, 0, len(issues))

	overdueCount, unassignedCount, homeCount, externalCount, highPriorityCount := 0, 0, 0, 0, 0
	oldestDays, totalAgeDays := 0, 0

	for _, inc := range issues {
		assignee := inc.Fields.Assignee.label()
		if inc.Fields.Assignee == nil {
			unassignedCount++
		}

		projectKey := inc.Fields.Project.Key
		scope := "external"
		if home[strings.ToUpper(projectKey)] {
			scope = "home"
			homeCount++
		} else {
			externalCount++
		}

		due := ""
		overdue := false
		if inc.Fields.DueDate != nil && *inc.Fields.DueDate != "" {
			due = *inc.Fields.DueDate
			if due < today {
				overdue = true
				overdueCount++
			}
		}

		priority := "Unset"
		if inc.Fields.Priority != nil && inc.Fields.Priority.Name != "" {
			priority = inc.Fields.Priority.Name
		}
		if isHighPriority(priority) {
			highPriorityCount++
		}

		created := ""
		ageDays := 0
		if t, ok := parseJiraTime(inc.Fields.Created); ok {
			created = t.Format("2006-01-02")
			ageDays = int(now.Sub(t).Hours() / 24)
			if ageDays > oldestDays {
				oldestDays = ageDays
			}
			totalAgeDays += ageDays
		}

		ag, ok := agMap[assignee]
		if !ok {
			ag = &agState{jsAssigneeGroup: jsAssigneeGroup{Name: assignee}, projectSet: map[string]bool{}}
			agMap[assignee] = ag
		}
		ag.Count++
		if overdue {
			ag.Overdue++
		}
		ag.projectSet[projectKey] = true

		projectMap[projectKey]++
		statusMap[inc.Fields.Status.Name]++
		priorityMap[priority]++

		jsIssues = append(jsIssues, jsIssue{
			Key:      inc.Key,
			Summary:  inc.Fields.Summary,
			Project:  projectKey,
			Scope:    scope,
			Assignee: assignee,
			Status:   inc.Fields.Status.Name,
			Priority: priority,
			Due:      due,
			Overdue:  overdue,
			Created:  created,
			AgeDays:  ageDays,
		})
	}

	assignees := make([]jsAssigneeGroup, 0, len(agMap))
	// namedTotal/namedCount exclude "Unassigned" — overload is a per-person
	// signal, and Unassigned isn't a person who can be overloaded.
	namedTotal, namedCount := 0, 0
	for name, ag := range agMap {
		for p := range ag.projectSet {
			ag.Projects = append(ag.Projects, p)
		}
		sort.Strings(ag.Projects)
		assignees = append(assignees, ag.jsAssigneeGroup)
		if name != "Unassigned" {
			namedTotal += ag.Count
			namedCount++
		}
	}
	sort.Slice(assignees, func(i, j int) bool { return assignees[i].Count > assignees[j].Count })

	teamAvg := 0.0
	if namedCount > 0 {
		teamAvg = float64(namedTotal) / float64(namedCount)
	}
	threshold := teamAvg * overloadMultiplier

	var callouts []jsCallout
	for _, ag := range assignees {
		if ag.Name == "Unassigned" || float64(ag.Count) < threshold || ag.Count < minOverloadCount {
			continue
		}
		pctAbove := 0
		if teamAvg > 0 {
			pctAbove = int(((float64(ag.Count) - teamAvg) / teamAvg) * 100)
		}
		callouts = append(callouts, jsCallout{
			Name:        ag.Name,
			Count:       ag.Count,
			TeamAvg:     math.Round(teamAvg*10) / 10,
			PctAboveAvg: pctAbove,
			Projects:    ag.Projects,
		})
	}

	oldestIssues := make([]jsIssue, len(jsIssues))
	copy(oldestIssues, jsIssues)
	sort.Slice(oldestIssues, func(i, j int) bool { return oldestIssues[i].AgeDays > oldestIssues[j].AgeDays })
	if len(oldestIssues) > oldestItemsLimit {
		oldestIssues = oldestIssues[:oldestItemsLimit]
	}

	avgAgeDays := 0
	if len(issues) > 0 {
		avgAgeDays = totalAgeDays / len(issues)
	}

	// Per-KPI JQL, each reusing the same filter base as the fetch query so a
	// dashboard card always opens the exact Jira view it summarizes.
	where, order := splitJQL(jql)
	if order == "" {
		order = " ORDER BY due ASC, created ASC"
	}
	homeJQL := fmt.Sprintf("%s AND project in (%s)%s", where, quoteJQLList(homeProjects), order)
	externalJQL := fmt.Sprintf("%s AND project not in (%s)%s", where, quoteJQLList(homeProjects), order)
	overdueJQL := fmt.Sprintf("%s AND due < now()%s", where, order)
	unassignedJQL := fmt.Sprintf("%s AND assignee is EMPTY%s", where, order)
	highPriorityJQL := fmt.Sprintf(`%s AND priority in ("Highest", "High")%s`, where, order)
	ageJQL := fmt.Sprintf("%s ORDER BY created ASC", where)

	mustJSON := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			log.Panicf("marshal: %v", err)
		}
		return string(b)
	}

	return dashData{
		FetchedAt:         now.Format("Jan 2 2006 15:04 MST"),
		BaseURL:           baseURL,
		BaseURLJSON:       mustJSON(baseURL),
		JQLEncoded:        url.QueryEscape(jql),
		TeamLabel:         strings.Join(members, ", "),
		HomeProjects:      strings.Join(homeProjects, ", "),
		TotalOpen:         len(issues),
		HomeCount:         homeCount,
		ExternalCount:     externalCount,
		OverdueCount:      overdueCount,
		UnassignedCount:   unassignedCount,
		ProjectCount:      len(projectMap),
		OldestDays:        oldestDays,
		AvgAgeDays:        avgAgeDays,
		HighPriorityCount: highPriorityCount,
		OverloadedCount:   len(callouts),
		AssigneesJSON:     mustJSON(assignees),
		ProjectsJSON:      mustJSON(projectMap),
		StatusJSON:        mustJSON(statusMap),
		PriorityJSON:      mustJSON(priorityMap),
		OldestJSON:        mustJSON(oldestIssues),
		CalloutsJSON:      mustJSON(callouts),
		IssuesJSON:        mustJSON(jsIssues),

		HomeJQLEncoded:         url.QueryEscape(homeJQL),
		ExternalJQLEncoded:     url.QueryEscape(externalJQL),
		OverdueJQLEncoded:      url.QueryEscape(overdueJQL),
		UnassignedJQLEncoded:   url.QueryEscape(unassignedJQL),
		HighPriorityJQLEncoded: url.QueryEscape(highPriorityJQL),
		AgeJQLEncoded:          url.QueryEscape(ageJQL),
	}
}

func openBrowser(absPath string) {
	url := "file://" + absPath
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	if err := exec.Command(cmd, args...).Start(); err != nil {
		log.Printf("open browser: %v", err)
	}
}
