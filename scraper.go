package main

import (
	"github.com/gocolly/colly"

	"os"

	"log"
)

func main() {
	f, err := os.Create("file.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	c := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"))
	b := colly.NewCollector(colly.AllowedDomains("courses.students.ubc.ca"))
	c.OnHTML("tr[class=section1], tr[class=section2]", func(h *colly.HTMLElement) {
		subject := h.ChildText("tr td:nth-of-type(1)")
		_, err := f.WriteString(subject + "\n")
		if err != nil {
			log.Fatal(err)
		}
		var coursePage string
		h.ForEach("a", func(_ int, e *colly.HTMLElement) {
			coursePage = e.Attr("href")
		})

		if coursePage != "" {

			b.Visit(h.Request.AbsoluteURL(coursePage))
			_, err := f.WriteString("Courses:\n")
			if err != nil {
				log.Fatal(err)
			}
			b.OnHTML("tr[class=section1], tr[class=section2]", func(e *colly.HTMLElement) {
				e.ForEach("a", func(_ int, e2 *colly.HTMLElement) {
					_, err := f.WriteString(e.Text + "\n")
					if err != nil {
						log.Fatal(err)
					}
				})

			})

		}

	})
	c.Visit("https://courses.students.ubc.ca/cs/courseschedule?pname=subjarea&tname=subj-all-departments")
	b.Visit("https://courses.students.ubc.ca/cs/courseschedule?pname=subjarea&tname=subj-all-departments")

}
