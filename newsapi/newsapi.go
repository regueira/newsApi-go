package newsapi

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

var _ NewsApi = (*newsApi)(nil)

var (
	defaultNewsApi = &newsApi{
		language:        "en",
		location:        "US",
		limit:           10,
		order:           false,
		client:          http.DefaultClient,
		defaultSelector: "", //No default
		ctx:             context.Background(),
	}

	googleNewsURL = url.URL{
		Scheme: "https",
		Host:   "news.google.com",
		Path:   "/",
	}
)

type newsApi struct {
	language        string
	location        string
	period          *time.Duration
	startDate       *time.Time
	endDate         *time.Time
	limit           int
	order           bool
	contentSelector map[string]string
	defaultSelector string
	client          *http.Client
	ctx             context.Context
}

func NewNewsApi(options ...NewsApiOption) *newsApi {
	n := defaultNewsApi

	for _, option := range options {
		option(n)
	}

	return n
}

// SetQueryOptions sets the query options
func (n *newsApi) SetQueryOptions(options ...QueryOption) {
	for _, option := range options {
		option(n)
	}
}

// GetTopNews gets the news by path and query
func (n *newsApi) GetTopNews() ([]*News, error) {
	return n.getNews("/rss", "")
}

// GetLocationNews gets the news by location
func (n *newsApi) GetLocationNews(location string) ([]*News, error) {
	if location == "" {
		return nil, ErrEmptyLocation
	}
	path := "rss/headlines/section/geo/" + location
	return n.getNews(path, "")
}

// GetTopicNews gets the news by topic
func (n *newsApi) GetTopicNews(topic string) ([]*News, error) {
	if topic == "" {
		return nil, ErrEmptyTopic
	}
	topic = strings.ToUpper(topic)
	if _, ok := TopicMap[topic]; !ok {
		return nil, ErrInvalidTopic
	}
	path := "rss/headlines/section/topic/" + topic
	return n.getNews(path, "")
}

// SearchNews searches the news by query
func (n *newsApi) SearchNews(query string) ([]*News, error) {
	if query == "" {
		return nil, ErrEmptyQuery
	}
	return n.getNews("rss/search", query)
}

// composeURL composes the url by path and query
func (n *newsApi) composeURL(path string, query string) url.URL {
	searchURL := googleNewsURL
	q := url.Values{}
	q.Add("hl", n.language)
	q.Add("gl", n.location)
	q.Add("ceid", n.location+":"+n.language)
	searchURL.Path = path
	if query != "" {
		q.Set("q", query)
		if n.period != nil {
			q.Set("q", q.Get("q")+" when:"+FormatDuration(*n.period))
		}
		if n.endDate != nil {
			q.Set("q", q.Get("q")+" before:"+n.endDate.Format(time.DateOnly))
		}
		if n.startDate != nil {
			q.Set("q", q.Get("q")+" after:"+n.startDate.Format(time.DateOnly))
		}
	}
	searchURL.RawQuery = q.Encode()
	return searchURL
}

// getNews gets the news by path and query
func (n *newsApi) getNews(path, query string) ([]*News, error) {
	searchURL := n.composeURL(path, query)
	req, err := http.NewRequest(http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("User-Agent", RandomUserAgent())

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	parser := gofeed.NewParser()
	feed, err := parser.ParseString(string(body))
	if err != nil {
		return nil, fmt.Errorf("error parsing response body: %w", err)
	}

	newsList := make([]*News, 0, len(feed.Items))

	for _, item := range feed.Items {
		news := NewNews(item)
		newsList = append(newsList, news)
	}
	// sort by published date
	if n.order {
		sort.Slice(newsList, func(i, j int) bool {
			return newsList[i].PublishedParsed.After(*newsList[j].PublishedParsed)
		})
	}
	// limit the number of news
	if n.limit > 0 && n.limit < len(newsList) {
		newsList = newsList[:n.limit]
	}
	return newsList, nil
}

// FetchSourceLinks fetches the source links by the google news links
func (n *newsApi) FetchSourceLinks(newsList []*News) {
	// create chrome browser context
	parentCtx, cancel := chromedp.NewContext(n.ctx)
	defer cancel()

	var wg sync.WaitGroup
	for _, news := range newsList {
		wg.Add(1)
		go func(news *News) {
			defer wg.Done()
			// create chrome tab context
			ctx, cancelTab := chromedp.NewContext(parentCtx)
			defer cancelTab()

			err := news.fetchSourceLink(ctx)
			if err != nil {
				log.Println(fmt.Printf("error fetching source link: %s", err))
			}
		}(news)
	}
	wg.Wait()
}

// FetchSourceContents fetches the source contents by the source links
func (n *newsApi) FetchSourceContents(newsList []*News) {
	var wg sync.WaitGroup
	for _, news := range newsList {
		wg.Add(1)
		go func(news *News) {
			defer wg.Done()

			linkURL, err := url.Parse(news.SourceLink)
			if err != nil {
				log.Println(fmt.Printf("error fetching source link: %s", err))
			} else {
				currSelector := n.contentSelector[linkURL.Host]
				if currSelector == "" && n.defaultSelector != "" {
					currSelector = n.defaultSelector
					log.Println("using default selector:", linkURL)
				}

				err = news.fetchSourceContent(currSelector)
				if err != nil {
					log.Println(fmt.Printf("error fetching source content: %s", err))
				}
			}
		}(news)
	}
	wg.Wait()
}
