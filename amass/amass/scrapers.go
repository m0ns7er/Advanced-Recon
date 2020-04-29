// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package amass

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ScraperService struct {
	BaseAmassService

	responses    chan *AmassRequest
	scrapers     []Scraper
	dnsdb        Scraper
	filter       map[string]struct{}
	domainFilter map[string]struct{}
}

func NewScraperService(config *AmassConfig) *ScraperService {
	ss := &ScraperService{
		responses:    make(chan *AmassRequest, 50),
		filter:       make(map[string]struct{}),
		domainFilter: make(map[string]struct{}),
	}

	ss.BaseAmassService = *NewBaseAmassService("Scraper Service", config, ss)
	ss.dnsdb = DNSDBScrape(ss.responses, config)
	ss.scrapers = []Scraper{
		AskScrape(ss.responses, config),
		BaiduScrape(ss.responses, config),
		BingScrape(ss.responses, config),
		CensysScrape(ss.responses, config),
		CertSpotterScrape(ss.responses, config),
		CertDBScrape(ss.responses, config),
		CrtshScrape(ss.responses, config),
		DogpileScrape(ss.responses, config),
		ss.dnsdb,
		DNSDumpsterScrape(ss.responses, config),
		ExaleadScrape(ss.responses, config),
		FindSubDomainsScrape(ss.responses, config),
		GoogleScrape(ss.responses, config),
		HackerTargetScrape(ss.responses, config),
		NetcraftScrape(ss.responses, config),
		PTRArchiveScrape(ss.responses, config),
		RiddlerScrape(ss.responses, config),
		RobtexScrape(ss.responses, config),
		SiteDossierScrape(ss.responses, config),
		ThreatCrowdScrape(ss.responses, config),
		ThreatMinerScrape(ss.responses, config),
		VirusTotalScrape(ss.responses, config),
		YahooScrape(ss.responses, config),
	}

	return ss
}

func (ss *ScraperService) OnStart() error {
	ss.BaseAmassService.OnStart()

	go ss.processOutput()
	go ss.executeAllScrapers()
	return nil
}

func (ss *ScraperService) OnStop() error {
	ss.BaseAmassService.OnStop()
	return nil
}

func (ss *ScraperService) processRequests() {
	done := make(chan int)

	t := time.NewTicker(ss.Config().Frequency)
	defer t.Stop()
loop:
	for {
		select {
		case <-t.C:
			if req := ss.NextRequest(); req != nil {
				ss.dnsdb.Scrape(req.Name, done)
				<-done
			}
		case <-ss.Quit():
			break loop
		}
	}
}

func (ss *ScraperService) processOutput() {
loop:
	for {
		select {
		case req := <-ss.responses:
			req.Name = strings.TrimSpace(trim252F(req.Name))

			if !ss.duplicate(req.Name) {
				ss.Config().dns.SendRequest(req)
			}
		case <-ss.Quit():
			break loop
		}
	}
}

// Returns true if the subdomain name is a duplicate entry in the filter.
// If not, the subdomain name is added to the filter
func (ss *ScraperService) duplicate(sub string) bool {
	if _, found := ss.filter[sub]; found {
		return true
	}
	ss.filter[sub] = struct{}{}
	return false
}

func (ss *ScraperService) executeAllScrapers() {
	done := make(chan int)

	ss.SetActive(true)
	// Loop over all the root domains provided in the config
	for _, domain := range ss.Config().Domains() {
		if _, found := ss.domainFilter[domain]; found {
			continue
		}
		// Kick off all the searches
		for _, s := range ss.scrapers {
			go s.Scrape(domain, done)
		}
		// Wait for them to complete
		for i := 0; i < len(ss.scrapers); i++ {
			<-done
		}
	}
	ss.SetActive(false)
}

// Searcher - represents all types that perform searches for domain names
type Scraper interface {
	Scrape(domain string, done chan int)
	fmt.Stringer
}

// searchEngine - A searcher that attempts to discover information using a web search engine
type searchEngine struct {
	Name     string
	Quantity int
	Limit    int
	Output   chan<- *AmassRequest
	Callback func(*searchEngine, string, int) string
	Config   *AmassConfig
}

func (se *searchEngine) String() string {
	return se.Name
}

func (se *searchEngine) urlByPageNum(domain string, page int) string {
	return se.Callback(se, domain, page)
}

func (se *searchEngine) Scrape(domain string, done chan int) {
	var unique []string

	re := SubdomainRegex(domain)
	num := se.Limit / se.Quantity
	for i := 0; i < num; i++ {
		page := GetWebPageWithDialContext(
			DialContext, se.urlByPageNum(domain, i), nil)
		if page == "" {
			break
		}

		for _, sd := range re.FindAllString(page, -1) {
			u := NewUniqueElements(unique, sd)

			if len(u) > 0 {
				unique = append(unique, u...)
				se.Output <- &AmassRequest{
					Name:   sd,
					Domain: domain,
					Tag:    SCRAPE,
					Source: se.Name,
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	done <- len(unique)
}

func askURLByPageNum(a *searchEngine, domain string, page int) string {
	p := strconv.Itoa(page)
	u, _ := url.Parse("https://www.ask.com/web")

	u.RawQuery = url.Values{"q": {"site:" + domain},
		"o": {"0"}, "l": {"dir"}, "qo": {"pagination"}, "page": {p}}.Encode()
	return u.String()
}

func AskScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &searchEngine{
		Name:     "Ask Scrape",
		Quantity: 10, // ask.com appears to be hardcoded at 10 results per page
		Limit:    100,
		Output:   out,
		Callback: askURLByPageNum,
		Config:   config,
	}
}

func baiduURLByPageNum(d *searchEngine, domain string, page int) string {
	pn := strconv.Itoa(page)
	u, _ := url.Parse("https://www.baidu.com/s")

	u.RawQuery = url.Values{"pn": {pn}, "wd": {domain}, "oq": {domain}}.Encode()
	return u.String()
}

func BaiduScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &searchEngine{
		Name:     "Baidu",
		Quantity: 20,
		Limit:    100,
		Output:   out,
		Callback: baiduURLByPageNum,
		Config:   config,
	}
}

func bingURLByPageNum(b *searchEngine, domain string, page int) string {
	count := strconv.Itoa(b.Quantity)
	first := strconv.Itoa((page * b.Quantity) + 1)
	u, _ := url.Parse("http://www.bing.com/search")

	u.RawQuery = url.Values{"q": {"domain:" + domain},
		"count": {count}, "first": {first}, "FORM": {"PORE"}}.Encode()
	return u.String()
}

func BingScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &searchEngine{
		Name:     "Bing Scrape",
		Quantity: 20,
		Limit:    200,
		Output:   out,
		Callback: bingURLByPageNum,
		Config:   config,
	}
}

func dogpileURLByPageNum(d *searchEngine, domain string, page int) string {
	qsi := strconv.Itoa(d.Quantity * page)
	u, _ := url.Parse("http://www.dogpile.com/search/web")

	u.RawQuery = url.Values{"qsi": {qsi}, "q": {domain}}.Encode()
	return u.String()
}

func DogpileScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &searchEngine{
		Name:     "Dogpile",
		Quantity: 15, // Dogpile returns roughly 15 results per page
		Limit:    90,
		Output:   out,
		Callback: dogpileURLByPageNum,
		Config:   config,
	}
}

func googleURLByPageNum(d *searchEngine, domain string, page int) string {
	start := strconv.Itoa(d.Quantity * page)
	u, _ := url.Parse("https://www.google.com/search")

	u.RawQuery = url.Values{
		"q":      {"site:" + domain},
		"btnG":   {"Search"},
		"hl":     {"en"},
		"biw":    {""},
		"bih":    {""},
		"gbv":    {"1"},
		"start":  {start},
		"filter": {"0"},
	}.Encode()
	return u.String()
}

func GoogleScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &searchEngine{
		Name:     "Google",
		Quantity: 10,
		Limit:    160,
		Output:   out,
		Callback: googleURLByPageNum,
		Config:   config,
	}
}

func yahooURLByPageNum(y *searchEngine, domain string, page int) string {
	b := strconv.Itoa(y.Quantity*page + 1)
	pz := strconv.Itoa(y.Quantity)

	u, _ := url.Parse("https://search.yahoo.com/search")
	u.RawQuery = url.Values{"p": {"site:" + domain},
		"b": {b}, "pz": {pz}, "bct": {"0"}, "xargs": {"0"}}.Encode()
	return u.String()
}

func YahooScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &searchEngine{
		Name:     "Yahoo",
		Quantity: 10,
		Limit:    100,
		Output:   out,
		Callback: yahooURLByPageNum,
		Config:   config,
	}
}

//--------------------------------------------------------------------------------------------
// lookup - A searcher that attempts to discover information on a single web page
type lookup struct {
	Name     string
	Output   chan<- *AmassRequest
	Callback func(string) string
	Config   *AmassConfig
}

func (l *lookup) String() string {
	return l.Name
}

func (l *lookup) Scrape(domain string, done chan int) {
	var unique []string

	re := SubdomainRegex(domain)
	page := GetWebPageWithDialContext(DialContext, l.Callback(domain), nil)
	if page == "" {
		done <- 0
		return
	}

	for _, sd := range re.FindAllString(page, -1) {
		u := NewUniqueElements(unique, sd)

		if len(u) > 0 {
			unique = append(unique, u...)
			l.Output <- &AmassRequest{
				Name:   sd,
				Domain: domain,
				Tag:    SCRAPE,
				Source: l.Name,
			}
		}
	}
	done <- len(unique)
}

func censysURL(domain string) string {
	format := "https://www.censys.io/domain/%s/table"

	return fmt.Sprintf(format, domain)
}

func CensysScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "Censys",
		Output:   out,
		Callback: censysURL,
		Config:   config,
	}
}

func certSpotterURL(domain string) string {
	format := "https://certspotter.com/api/v0/certs?domain=%s"

	return fmt.Sprintf(format, domain)
}

func CertSpotterScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "CertSpotter",
		Output:   out,
		Callback: certSpotterURL,
		Config:   config,
	}
}

func dnsdbURL(domain string) string {
	format := "http://www.dnsdb.org/%s/"

	return fmt.Sprintf(format, domain)
}

func DNSDBScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "DNSDB",
		Output:   out,
		Callback: dnsdbURL,
		Config:   config,
	}
}

func exaleadURL(domain string) string {
	base := "http://www.exalead.com/search/web/results/"
	format := base + "?q=site:%s+-www?elements_per_page=50"

	return fmt.Sprintf(format, domain)
}

func ExaleadScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "Exalead",
		Output:   out,
		Callback: exaleadURL,
		Config:   config,
	}
}

func findSubDomainsURL(domain string) string {
	format := "https://findsubdomains.com/subdomains-of/%s"

	return fmt.Sprintf(format, domain)
}

func FindSubDomainsScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "FindSubDmns",
		Output:   out,
		Callback: findSubDomainsURL,
		Config:   config,
	}
}

func hackertargetURL(domain string) string {
	format := "http://api.hackertarget.com/hostsearch/?q=%s"

	return fmt.Sprintf(format, domain)
}

func HackerTargetScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "HackerTargt",
		Output:   out,
		Callback: hackertargetURL,
		Config:   config,
	}
}

func netcraftURL(domain string) string {
	format := "https://searchdns.netcraft.com/?restriction=site+ends+with&host=%s"

	return fmt.Sprintf(format, domain)
}

func NetcraftScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "Netcraft",
		Output:   out,
		Callback: netcraftURL,
		Config:   config,
	}
}

func ptrArchiveURL(domain string) string {
	format := "http://ptrarchive.com/tools/search2.htm?label=%s&date=ALL"

	return fmt.Sprintf(format, domain)
}

func PTRArchiveScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "PTRarchive",
		Output:   out,
		Callback: ptrArchiveURL,
		Config:   config,
	}
}

func riddlerURL(domain string) string {
	format := "https://riddler.io/search?q=pld:%s"

	return fmt.Sprintf(format, domain)
}

func RiddlerScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "Riddler",
		Output:   out,
		Callback: riddlerURL,
		Config:   config,
	}
}

func siteDossierURL(domain string) string {
	format := "http://www.sitedossier.com/parentdomain/%s"

	return fmt.Sprintf(format, domain)
}

func SiteDossierScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "SiteDossier",
		Output:   out,
		Callback: siteDossierURL,
		Config:   config,
	}
}

func threatCrowdURL(domain string) string {
	format := "https://www.threatcrowd.org/searchApi/v2/domain/report/?domain=%s"

	return fmt.Sprintf(format, domain)
}

func ThreatCrowdScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "ThreatCrowd",
		Output:   out,
		Callback: threatCrowdURL,
		Config:   config,
	}
}

func threatMinerURL(domain string) string {
	format := "https://www.threatminer.org/getData.php?e=subdomains_container&q=%s&t=0&rt=10&p=1"

	return fmt.Sprintf(format, domain)
}

func ThreatMinerScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "ThreatMiner",
		Output:   out,
		Callback: threatMinerURL,
		Config:   config,
	}
}

func virusTotalURL(domain string) string {
	format := "https://www.virustotal.com/en/domain/%s/information/"

	return fmt.Sprintf(format, domain)
}

func VirusTotalScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &lookup{
		Name:     "VirusTotal",
		Output:   out,
		Callback: virusTotalURL,
		Config:   config,
	}
}

//--------------------------------------------------------------------------------------------

type robtex struct {
	Name   string
	Base   string
	Output chan<- *AmassRequest
	Config *AmassConfig
}

func (r *robtex) String() string {
	return r.Name
}

type robtexJSON struct {
	Name string `json:"rrname"`
	Data string `json:"rrdata"`
	Type string `json:"rrtype"`
}

func (r *robtex) Scrape(domain string, done chan int) {
	var ips []string
	var unique []string

	page := GetWebPageWithDialContext(
		DialContext, r.Base+"forward/"+domain, nil)
	if page == "" {
		done <- 0
		return
	}

	lines := r.parseJSON(page)
	for _, line := range lines {
		if line.Type == "A" {
			ips = UniqueAppend(ips, line.Data)
		}
	}

	var list string
	for _, ip := range ips {
		time.Sleep(500 * time.Millisecond)

		pdns := GetWebPageWithDialContext(
			DialContext, r.Base+"reverse/"+ip, nil)
		if pdns == "" {
			continue
		}

		rev := r.parseJSON(pdns)
		for _, line := range rev {
			list += line.Name + " "
		}
	}

	re := SubdomainRegex(domain)
	for _, sd := range re.FindAllString(list, -1) {
		u := NewUniqueElements(unique, sd)

		if len(u) > 0 {
			unique = append(unique, u...)
			r.Output <- &AmassRequest{
				Name:   sd,
				Domain: domain,
				Tag:    SCRAPE,
				Source: r.Name,
			}
		}
	}
	done <- len(unique)
}

func (r *robtex) parseJSON(page string) []robtexJSON {
	var lines []robtexJSON

	scanner := bufio.NewScanner(strings.NewReader(page))
	for scanner.Scan() {
		// Get the next line of JSON
		line := scanner.Text()
		if line == "" {
			continue
		}

		var j robtexJSON

		err := json.Unmarshal([]byte(line), &j)
		if err != nil {
			continue
		}

		lines = append(lines, j)
	}
	return lines
}

func RobtexScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &robtex{
		Name:   "Robtex",
		Base:   "https://freeapi.robtex.com/pdns/",
		Output: out,
		Config: config,
	}
}

//--------------------------------------------------------------------------------------------

type dumpster struct {
	Name   string
	Base   string
	Output chan<- *AmassRequest
	Config *AmassConfig
}

func (d *dumpster) String() string {
	return d.Name
}

func (d *dumpster) Scrape(domain string, done chan int) {
	var unique []string

	page := GetWebPageWithDialContext(DialContext, d.Base, nil)
	if page == "" {
		done <- 0
		return
	}

	token := d.getCSRFToken(page)
	if token == "" {
		done <- 0
		return
	}

	page = d.postForm(token, domain)
	if page == "" {
		done <- 0
		return
	}

	re := SubdomainRegex(domain)
	for _, sd := range re.FindAllString(page, -1) {
		u := NewUniqueElements(unique, sd)

		if len(u) > 0 {
			unique = append(unique, u...)
			d.Output <- &AmassRequest{
				Name:   sd,
				Domain: domain,
				Tag:    SCRAPE,
				Source: d.Name,
			}
		}
	}
	done <- len(unique)
}

func (d *dumpster) getCSRFToken(page string) string {
	re := regexp.MustCompile("<input type='hidden' name='csrfmiddlewaretoken' value='([a-zA-Z0-9]*)' />")

	if subs := re.FindStringSubmatch(page); len(subs) == 2 {
		return strings.TrimSpace(subs[1])
	}
	return ""
}

func (d *dumpster) postForm(token, domain string) string {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext:         DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	params := url.Values{
		"csrfmiddlewaretoken": {token},
		"targetip":            {domain},
	}

	req, err := http.NewRequest("POST", d.Base, strings.NewReader(params.Encode()))
	if err != nil {
		return ""
	}
	// The CSRF token needs to be sent as a cookie
	cookie := &http.Cookie{
		Name:   "csrftoken",
		Domain: "dnsdumpster.com",
		Value:  token,
	}
	req.AddCookie(cookie)

	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("Accept", ACCEPT)
	req.Header.Set("Accept-Language", ACCEPT_LANG)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://dnsdumpster.com")
	req.Header.Set("X-CSRF-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	// Now, grab the entire page
	in, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return string(in)
}

func DNSDumpsterScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &dumpster{
		Name:   "DNSDumpster",
		Base:   "https://dnsdumpster.com/",
		Output: out,
		Config: config,
	}
}

//--------------------------------------------------------------------------------------------

type crtsh struct {
	Name   string
	Base   string
	Output chan<- *AmassRequest
	Config *AmassConfig
}

func (c *crtsh) String() string {
	return c.Name
}

func (c *crtsh) Scrape(domain string, done chan int) {
	var unique []string

	// Pull the page that lists all certs for this domain
	page := GetWebPageWithDialContext(DialContext, c.Base+"?q=%25."+domain, nil)
	if page == "" {
		done <- 0
		return
	}
	// Get the subdomain name the cert was issued to, and
	// the Subject Alternative Name list from each cert
	results := c.getSubmatches(page)
	for _, rel := range results {
		// Do not go too fast
		time.Sleep(50 * time.Millisecond)
		// Pull the certificate web page
		cert := GetWebPageWithDialContext(DialContext, c.Base+rel, nil)
		if cert == "" {
			continue
		}
		// Get all names off the certificate
		unique = UniqueAppend(unique, c.getMatches(cert, domain)...)
	}
	if len(unique) > 0 {
		c.sendAllNames(unique, domain)
	}
	done <- len(unique)
}

func (c *crtsh) sendAllNames(names []string, domain string) {
	for _, name := range names {
		c.Output <- &AmassRequest{
			Name:   name,
			Domain: domain,
			Tag:    SCRAPE,
			Source: c.Name,
		}
	}
}

func (c *crtsh) getMatches(content, domain string) []string {
	var results []string

	re := SubdomainRegex(domain)
	for _, s := range re.FindAllString(content, -1) {
		results = append(results, s)
	}
	return results
}

func (c *crtsh) getSubmatches(content string) []string {
	var results []string

	re := regexp.MustCompile("<TD style=\"text-align:center\"><A href=\"([?]id=[a-zA-Z0-9]*)\">[a-zA-Z0-9]*</A></TD>")
	for _, subs := range re.FindAllStringSubmatch(content, -1) {
		results = append(results, strings.TrimSpace(subs[1]))
	}
	return results
}

// CrtshSearch - A searcher that attempts to discover names from SSL certificates
func CrtshScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &crtsh{
		Name:   "crt.sh",
		Base:   "https://crt.sh/",
		Output: out,
		Config: config,
	}
}

//--------------------------------------------------------------------------------------------

type certdb struct {
	Name   string
	Base   string
	Output chan<- *AmassRequest
	Config *AmassConfig
}

func (c *certdb) String() string {
	return c.Name
}

func (c *certdb) Scrape(domain string, done chan int) {
	var unique []string

	// Pull the page that lists all certs for this domain
	page := GetWebPageWithDialContext(DialContext, c.Base+"/domain/"+domain, nil)
	if page == "" {
		done <- 0
		return
	}
	// Get the subdomain name the cert was issued to, and
	// the Subject Alternative Name list from each cert
	results := c.getSubmatches(page)
	for _, rel := range results {
		// Do not go too fast
		time.Sleep(50 * time.Millisecond)
		// Pull the certificate web page
		cert := GetWebPageWithDialContext(DialContext, c.Base+rel, nil)
		if cert == "" {
			continue
		}
		// Get all names off the certificate
		unique = UniqueAppend(unique, c.getMatches(cert, domain)...)
	}
	if len(unique) > 0 {
		c.sendAllNames(unique, domain)
	}
	done <- len(unique)
}

func (c *certdb) sendAllNames(names []string, domain string) {
	for _, name := range names {
		c.Output <- &AmassRequest{
			Name:   name,
			Domain: domain,
			Tag:    SCRAPE,
			Source: c.Name,
		}
	}
}

func (c *certdb) getMatches(content, domain string) []string {
	var results []string

	re := SubdomainRegex(domain)
	for _, s := range re.FindAllString(content, -1) {
		results = append(results, s)
	}
	return results
}

func (c *certdb) getSubmatches(content string) []string {
	var results []string

	re := regexp.MustCompile("<a href=\"(/ssl-cert/[a-zA-Z0-9]*)\" class=\"see-more-link\">")
	for _, subs := range re.FindAllStringSubmatch(content, -1) {
		results = append(results, strings.TrimSpace(subs[1]))
	}
	return results
}

// CrtshSearch - A searcher that attempts to discover names from SSL certificates
func CertDBScrape(out chan<- *AmassRequest, config *AmassConfig) Scraper {
	return &certdb{
		Name:   "CertDB",
		Base:   "https://certdb.com",
		Output: out,
		Config: config,
	}
}
