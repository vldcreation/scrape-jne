package main

import (
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func scrapeShippingFee(url string) (originLabel, destLabel string, tariff []TariffEntry, err error) {
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		RandomDelay: 1 * time.Second,
	})

	done := make(chan struct{})

	c.OnHTML("div.box-tujuan", func(e *colly.HTMLElement) {
		e.ForEach("div.box-tujuan__list", func(_ int, el *colly.HTMLElement) {
			h6 := strings.TrimSpace(el.ChildText("h6"))
			p := strings.TrimSpace(el.ChildText("p"))
			switch h6 {
			case "Dari":
				originLabel = p
			case "Tujuan":
				destLabel = p
			}
		})
	})

	c.OnHTML("div.wrap-table table tbody tr", func(e *colly.HTMLElement) {
		service := strings.TrimSpace(e.ChildText("td:nth-child(1)"))
		shipmentType := strings.TrimSpace(e.ChildText("td:nth-child(2)"))
		fee := strings.TrimSpace(e.ChildText("td:nth-child(3)"))
		etd := strings.TrimSpace(e.ChildText("td:nth-child(4)"))

		tariff = append(tariff, TariffEntry{
			ServiceName:  service,
			ShipmentType: shipmentType,
			Fee:          fee,
			ETD:          etd,
		})
	})

	c.OnError(func(r *colly.Response, e error) {
		err = e
		close(done)
	})

	c.OnScraped(func(r *colly.Response) {
		close(done)
	})

	c.Visit(url)
	c.Wait()
	<-done

	return originLabel, destLabel, tariff, err
}
