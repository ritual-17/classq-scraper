package main

import (
	"strings"

	"github.com/gocolly/colly"

	"os"

	"log"
)

type Section struct {
	status   string
	section  string
	activity string
	term     string
	days     string
	start    string
	end      string
}

func main() {
	f, err := os.Create("file.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	subjects := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"))
	courses := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"))
	sections := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"))
	seats := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"))

	seats.OnHTML("tbody", func(h *colly.HTMLElement) {

		firstRow := h.ChildText("tr td:nth-of-type(1)")
		if !strings.Contains(firstRow, "Total Seats Remaining:") {
			return
		}

		seatsRemaining := h.DOM.Children().Eq(0).Text()
		f.WriteString(seatsRemaining + "\n")

	})

	sections.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		section := Section{
			status:   h.ChildText("tr td:nth-of-type(1)"),
			section:  h.ChildText("tr td:nth-of-type(2)"),
			activity: h.ChildText("tr td:nth-of-type(3)"),
			term:     h.ChildText("tr td:nth-of-type(4)"),
			days:     h.ChildText("tr td:nth-of-type(7)"),
			start:    h.ChildText("tr td:nth-of-type(8)"),
			end:      h.ChildText("tr td:nth-of-type(9)"),
		}

		if section.status == "" {
			section.status = "Open"
		}
		f.WriteString("Section:\n")
		f.WriteString(section.status + "\n")
		f.WriteString(section.section + "\n")
		f.WriteString(section.activity + "\n")
		f.WriteString(section.term + "\n")
		f.WriteString(section.days + "\n")
		f.WriteString(section.start + "\n")
		f.WriteString(section.end + "\n")

		var seatsPage string
		h.ForEach("a", func(_ int, e *colly.HTMLElement) {
			seatsPage = e.Attr("href")
		})

		if seatsPage == "" {
			return
		}

		seats.Visit(h.Request.AbsoluteURL(seatsPage))

	})

	//course page scraper that finds all courses
	courses.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		var sectionsPage string
		h.ForEach("a", func(_ int, e *colly.HTMLElement) {
			_, err := f.WriteString(e.Text + "\n")
			if err != nil {
				log.Fatal(err)
			}
			sectionsPage = e.Attr("href")
		})

		if sectionsPage == "" {
			return
		}

		sections.Visit(h.Request.AbsoluteURL(sectionsPage))

	})

	count := 0

	//main page scraper that finds all subjects
	subjects.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		count++
		//for testing, only looking at Anthropology
		if count != 8 {
			return
		}

		subject := h.ChildText("tr td:nth-of-type(1)")
		_, err := f.WriteString(subject + "\n")
		if err != nil {
			log.Fatal(err)
		}
		var coursePage string
		h.ForEach("a", func(_ int, e *colly.HTMLElement) {
			coursePage = e.Attr("href")
		})

		if coursePage == "" {
			return
		}
		_, err = f.WriteString("COURSES:\n")
		if err != nil {
			log.Fatal(err)
		}

		courses.Visit(h.Request.AbsoluteURL(coursePage))

	})
	subjects.Visit("https://courses.students.ubc.ca/cs/courseschedule?pname=subjarea&tname=subj-all-departments")

}
