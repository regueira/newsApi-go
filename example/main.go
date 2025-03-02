package main

import (
	"fmt"
	"time"

	"github.com/regueira/newsApi-go/newsapi"
)

func main() {
	handler := newsapi.NewNewsApi()

	queryOptions := []newsapi.QueryOption{}
	queryOptions = append(queryOptions, newsapi.WithLanguage(newsapi.LanguageSpanish))
	queryOptions = append(queryOptions, newsapi.WithLocation(newsapi.LocationArgentina))
	queryOptions = append(queryOptions, newsapi.WithLimit(10))
	//endDate := time.Now()
	//startDate := endDate.Add(-time.Hour * 72)
	//queryOptions = append(queryOptions, newsapi.WithStartDate(startDate))
	//queryOptions = append(queryOptions, newsapi.WithEndDate(endDate))
	queryOptions = append(queryOptions, newsapi.WithPeriod(time.Hour*2))
	queryOptions = append(queryOptions, newsapi.WithDefaultSelector("body"))
	queryOptions = append(queryOptions, newsapi.WithContentSelector(map[string]string{
		"tn.com.ar":           ".col-content",
		"www.lanacion.com.ar": ".cuerpo__nota",
		"www.clarin.com":      "#storyBody",
		"www.pagina12.com.ar": ".article-main-content",
		"www.minutouno.com":   ".detail-body",
		"chequeado.com":       ".c-nota__content-from-editor",
		"eleconomista.com.ar": ".content",
		"rionegro.com.ar":     ".newsfull__body",
		"www.0223.com.ar":     ".cont_cuerpo",
		"www.infobae.com":     ".body-article",
		"www.eldiarioar.com":  ".c-content",
	}))
	handler.SetQueryOptions(queryOptions...)

	newsList, err := handler.SearchNews("Politica")
	if err != nil {
		fmt.Println(err)
		return
	}
	handler.FetchSourceLinks(newsList)
	handler.FetchSourceContents(newsList)
	for _, news := range newsList {
		fmt.Println("=================================")
		fmt.Println(news.Title)
		fmt.Println(news.Link)

		fmt.Println(news.SourceLink)
		fmt.Println(news.SourceTitle)
		fmt.Println(news.SourceImageURL)
		fmt.Println(news.SourceImageWidth)
		fmt.Println(news.SourceImageHeight)
		fmt.Println(news.SourceDescription)
		fmt.Println(news.SourceKeywords)
		fmt.Println(news.SourceSiteName)
		fmt.Println(news.SourceIconUrl)
		fmt.Println(news.SourceContent)
	}
}
