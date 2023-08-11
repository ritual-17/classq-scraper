package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"os"
)

type Courses struct {
	Courses []Course `json:"courses"`
}

type Course struct {
	Name     string    `json:"name"`
	Sections []Section `json:"sections"`
}

type Section struct {
	Name           string `json:"name"`
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

var errors []string = []string{}

var e = 0

var c *colly.Collector = colly.NewCollector(
	colly.AllowedDomains("courses.students.ubc.ca"),
	colly.Async(true),
	colly.CheckHead(),
)

func main() {

	defer timer()()

	StartScraping()

	courses := Courses{Courses: courseList}

	coursesJson, err := json.Marshal(courses)
	if err != nil {
		fmt.Print(err)
	}

	os.WriteFile("courses.json", coursesJson, 0644)

	fmt.Println("Errors:", errors)
	fmt.Println("Number of errors:", len(errors))

}

func timer() func() {
	start := time.Now()
	return func() {
		fmt.Printf("Time elapsed: %v", time.Since(start))
	}
}

func errorCheck() {
	e++
	if e > 10 {
		fmt.Println("10 ERRORS, sleeping for 5 seconds------------------------------------------------------------------------")
		time.Sleep(5 * time.Second)
		e = 0
	}
}

func StartScraping() {
	subjects := c.Clone()
	count := 0

	subjects.Limit(&colly.LimitRule{
		Delay: 10 * time.Second,
	})

	//main page scraper that finds all subjects
	subjects.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		count++
		//course.subject = h.ChildText("tr td:nth-of-type(1)")
		//fmt.Println("Now visiting: " + h.ChildText("tr td:nth-of-type(1)"))

		if count > 100 {
			return
		}

		var coursePage string
		h.ForEach("a", func(_ int, e *colly.HTMLElement) {
			coursePage = e.Attr("href")
		})

		if coursePage != "" {
			ScrapeCoursePage(h.Request.AbsoluteURL(coursePage))
		}

	})

	subjects.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error on Subjects page: ", r, "------", err, "----------")
		errorCheck()
	})

	subjects.Visit("https://courses.students.ubc.ca/cs/courseschedule?pname=subjarea&tname=subj-all-departments")
	subjects.Wait()

}

func ScrapeCoursePage(url string) {

	courses := c.Clone()

	courses.Limit(&colly.LimitRule{
		Parallelism: 2,
		Delay:       10 * time.Second,
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

	courses.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error on Courses page: ", r.Request.URL, "----------------")
		errorCheck()
	})

	courses.Visit(url)
	courses.Wait()

}

func ScrapeSectionPage(url string, course Course) Course {
	time.Sleep(250 * time.Millisecond)
	sections := c.Clone()

	sections.Limit(&colly.LimitRule{
		Parallelism: 2,
		Delay:       10 * time.Second,
	})

	sections.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		var section Section
		section.Status = h.ChildText("tr td:nth-of-type(1)")
		section.Name = h.ChildText("tr td:nth-of-type(2)")
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
		updatedSection := ScrapeSeatsPage(section.SeatsURL, section, course)

		success := false

		//retries 3 times if the request fails
		for i := 1; i < 4; i++ {
			if updatedSection.SeatsRemaining != "Error" {
				if i != 1 {
					fmt.Println("Success:", section.Name)
				}
				success = true
				break
			}
			fmt.Println("Retry number:", i)
			time.Sleep(1 * time.Second)
			updatedSection = ScrapeSeatsPage(section.SeatsURL, section, course)
		}

		if !success {
			errors = append(errors, section.Name)
			fmt.Println("Failed:", section.Name)
		}

		course.Sections = append(course.Sections, updatedSection)
	})

	sections.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error on Sections page for: ", course.Name, "----------------")
		errorCheck()
	})
	sections.Visit(url)
	sections.Wait()

	return course
}

func ScrapeSeatsPage(url string, section Section, course Course) Section {
	time.Sleep(250 * time.Millisecond)
	seats := c.Clone()

	seats.Limit(&colly.LimitRule{
		Parallelism: 2,
		Delay:       10 * time.Second,
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
		//fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
		fmt.Println("Error on Seats page for:", section.Name, "----------------")
		section.SeatsRemaining = "Error"
		errorCheck()
	})

	seats.Visit(url)
	seats.Wait()

	return section
}
