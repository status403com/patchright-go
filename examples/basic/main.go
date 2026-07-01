package main

import (
	"fmt"
	"log"

	patchright "github.com/status403com/patchright-go"
)

func main() {
	pw, err := patchright.Run()
	if err != nil {
		log.Fatalf("could not start patchright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(patchright.BrowserTypeLaunchOptions{
		Headless: patchright.Bool(false),
	})
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewStealthPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}

	if _, err = page.Goto("https://example.com"); err != nil {
		log.Fatalf("could not goto: %v", err)
	}

	title, err := page.Title()
	if err != nil {
		log.Fatalf("could not get title: %v", err)
	}
	fmt.Printf("Title: %s\n", title)

	webdriver, err := page.Evaluate("() => navigator.webdriver")
	if err != nil {
		log.Fatalf("could not evaluate: %v", err)
	}
	fmt.Printf("navigator.webdriver: %v\n", webdriver)
}
