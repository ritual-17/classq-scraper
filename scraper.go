package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly"

	"os"

	"log"
)

type Section struct {
	subject        string
	course         string
	status         string
	section        string
	activity       string
	instructor     string
	term           string
	days           string
	start          string
	end            string
	seatsURL       string
	seatsRemaining string
}

var sections []Section = []Section{}

func main() {

	start := time.Now()

	f, err := os.Create("file.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	StartScraping()

	for _, section := range sections {
		f.WriteString("Subject: " + section.subject + "\n")
		f.WriteString("Course: " + section.course + "\n")
		f.WriteString("Instructor: " + section.instructor + "\n")
		f.WriteString("Status: " + section.status + "\n")
		f.WriteString("Section: " + section.section + "\n")
		f.WriteString("Type: " + section.activity + "\n")
		f.WriteString("Term: " + section.term + "\n")
		f.WriteString("Days: " + section.days + "\n")
		f.WriteString("Start Time: " + section.start + "\n")
		f.WriteString("End Time: " + section.end + "\n")
		f.WriteString("URL: " + section.seatsURL + "\n")
		f.WriteString("Seats Remaining: " + section.seatsRemaining + "\n")
		f.WriteString("-----------------------------------\n")
	}

	elapsed := time.Since(start)
	fmt.Printf("Time elapsed: %s", elapsed)

}

func StartScraping() {
	subjects := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"), colly.Async(true))
	count := 0

	subjects.Limit(&colly.LimitRule{
		Delay: 10 * time.Second,
	})

	//main page scraper that finds all subjects
	subjects.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		var section Section
		count++
		//for testing, only looking at Anthropology
		if count != 8 {
			return
		}

		section.subject = h.ChildText("tr td:nth-of-type(1)")

		var coursePage string
		h.ForEach("a", func(_ int, e *colly.HTMLElement) {
			coursePage = e.Attr("href")
		})

		if coursePage == "" {
			return
		}

		ScrapeCoursePage(h.Request.AbsoluteURL(coursePage), section)

	})
	subjects.Visit("https://courses.students.ubc.ca/cs/courseschedule?pname=subjarea&tname=subj-all-departments")
	subjects.Wait()

}

func ScrapeCoursePage(url string, section Section) {
	courses := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"))

	courses.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		var sectionsPage string
		h.ForEach("a", func(_ int, e *colly.HTMLElement) {
			section.course = e.Text
			sectionsPage = e.Attr("href")
		})

		if sectionsPage == "" {
			return
		}

		ScrapeSectionPage(h.Request.AbsoluteURL(sectionsPage), section)

	})

	courses.Visit(url)

}

func ScrapeSectionPage(url string, section Section) {
	sections := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"))

	sections.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {

		section.status = h.ChildText("tr td:nth-of-type(1)")
		section.section = h.ChildText("tr td:nth-of-type(2)")
		section.activity = h.ChildText("tr td:nth-of-type(3)")
		section.term = h.ChildText("tr td:nth-of-type(4)")
		section.days = h.ChildText("tr td:nth-of-type(7)")
		section.start = h.ChildText("tr td:nth-of-type(8)")
		section.end = h.ChildText("tr td:nth-of-type(9)")

		if section.status == "" {
			section.status = "Open"
		}

		seatsPage := h.ChildAttr("tr td:nth-of-type(2) a", "href")

		if seatsPage == "" {
			return
		}

		section.seatsURL = h.Request.AbsoluteURL(seatsPage)
		ScrapeSeatsPage(section.seatsURL, section)

	})

	sections.Visit(url)

}

func ScrapeSeatsPage(url string, section Section) {
	seats := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"), colly.Async(true))

	seats.Limit(&colly.LimitRule{
		Delay: 10 * time.Second,
	})

	seats.OnHTML("tbody", func(h *colly.HTMLElement) {

		firstRow := h.ChildText("tr td:nth-of-type(1)")

		if strings.Contains(firstRow, "Total Seats Remaining:") {
			seatsRemaining := h.DOM.Children().Eq(0).Text()
			section.seatsRemaining = seatsRemaining
		} else if strings.Contains(firstRow, "Instructor:") {
			instructor := h.DOM.Children().Eq(0).Children().Eq(1).Text()
			section.instructor = instructor
		}

	})

	seats.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	seats.Visit(url)
	seats.Wait()

	sections = append(sections, section)
}
