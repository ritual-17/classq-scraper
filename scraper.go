package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly"

	"os"
)

type Course struct {
	Name     string    `json:"name"`
	Sections []Section `json:"sections"`
}

type Section struct {
	Section        string `json:"section"`
	Status         string `json:"status"`
	Activity       string `json:"activity"`
	Instructor     string `json:"instructor"`
	Term           string `json:"term"`
	Days           string `json:"days"`
	Start          string `json:"start"`
	End            string `json:"end"`
	SeatsURL       string `json:"seatsURL"`
	SeatsRemaining string `json:"seatsRemaining"`
}

var courseList []Course = []Course{}

func main() {

	start := time.Now()

	StartScraping()

	courses, err := json.Marshal(courseList)
	if err != nil {
		fmt.Print(err)
	}

	os.WriteFile("courses.json", courses, 0644)

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
		count++

		//look at every 20th subject for testing purposes
		if count%20 != 0 {
			return
		}

		//course.subject = h.ChildText("tr td:nth-of-type(1)")
		fmt.Println("Now visiting: " + h.ChildText("tr td:nth-of-type(1)"))

		var coursePage string
		h.ForEach("a", func(_ int, e *colly.HTMLElement) {
			coursePage = e.Attr("href")
		})

		if coursePage == "" {
			return
		}

		ScrapeCoursePage(h.Request.AbsoluteURL(coursePage))

	})
	subjects.Visit("https://courses.students.ubc.ca/cs/courseschedule?pname=subjarea&tname=subj-all-departments")
	subjects.Wait()

}

func ScrapeCoursePage(url string) {
	courses := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"), colly.Async(true))

	courses.Limit(&colly.LimitRule{
		Delay: 10 * time.Second,
	})

	courses.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		var course Course
		course.Sections = []Section{}
		var sectionsPage string

		h.ForEach("a", func(_ int, e *colly.HTMLElement) {
			course.Name = e.Text
			sectionsPage = e.Attr("href")
		})

		if sectionsPage == "" {
			return
		}

		courseList = append(courseList, ScrapeSectionPage(h.Request.AbsoluteURL(sectionsPage), course))

	})

	courses.Visit(url)
	courses.Wait()

}

func ScrapeSectionPage(url string, course Course) Course {
	sections := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"), colly.Async(true))

	sections.Limit(&colly.LimitRule{
		Delay: 10 * time.Second,
	})

	sections.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		var section Section
		section.Status = h.ChildText("tr td:nth-of-type(1)")
		section.Section = h.ChildText("tr td:nth-of-type(2)")
		section.Activity = h.ChildText("tr td:nth-of-type(3)")
		section.Term = h.ChildText("tr td:nth-of-type(4)")
		section.Days = h.ChildText("tr td:nth-of-type(7)")
		section.Start = h.ChildText("tr td:nth-of-type(8)")
		section.End = h.ChildText("tr td:nth-of-type(9)")

		if section.Status == "" {
			section.Status = "Open"
		}

		seatsPage := h.ChildAttr("tr td:nth-of-type(2) a", "href")

		if seatsPage == "" {
			return
		}

		section.SeatsURL = h.Request.AbsoluteURL(seatsPage)
		course.Sections = append(course.Sections, ScrapeSeatsPage(section.SeatsURL, section, course))
	})

	sections.Visit(url)
	sections.Wait()

	return course
}

func ScrapeSeatsPage(url string, section Section, course Course) Section {
	seats := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"), colly.Async(true))

	seats.Limit(&colly.LimitRule{
		Delay: 10 * time.Second,
	})

	seats.OnHTML("tbody", func(h *colly.HTMLElement) {

		firstRow := h.ChildText("tr td:nth-of-type(1)")

		if strings.Contains(firstRow, "Total Seats Remaining:") {
			seatsRemaining := h.DOM.Children().Eq(0).Text()
			section.SeatsRemaining = seatsRemaining
		} else if strings.Contains(firstRow, "Instructor:") {
			instructor := h.DOM.Children().Eq(0).Children().Eq(1).Text()
			section.Instructor = instructor
		}

	})

	seats.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	seats.Visit(url)
	seats.Wait()

	return section
}
