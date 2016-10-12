/**
 * WordPress Tickets
 * https://cixtor.com/
 * https://github.com/cixtor/wptickets
 * https://codex.wordpress.org/Using_the_Support_Forums
 * https://wordpress.org/support/
 *
 * Visualize the status of multiple support requests for a WordPress plugin.
 *
 * The WordPress Support Forums are a fantastic resource with a ton of
 * information, but sometimes people have trouble getting help there and they
 * don't know why. This is usually the result of a communication gap. The
 * WordPress forums have one of the most helpful communities on the web, you
 * just need to help them help you. Note: Please read the Supported Versions
 * information as the the WordPress Support Forums only provide assistance for
 * officially released versions of WordPress.
 *
 * This tools sends HTTP requests to the latest twenty pages of the support page
 * of the specified plugin, finds how many of the tickets are marked as
 * resolved, and shows which ones are missing. Many people in the WordPress
 * community sees the number of resolved tickets per month as a sign of
 * responsiveness and promptness, other use this as one of the main reasons to
 * install or not a plugin.
 */

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

func httpRequest(urlStr string) []byte {
	req, err := http.NewRequest("GET", urlStr, nil)

	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("dnt", "1")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("authority", "wordpress.org")
	req.Header.Set("accept-language", "en-US,en")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (KHTML, like Gecko) Safari/537.36")
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	return body
}

func analyzeMonthStats(plugin string) {
	var urlStr string = fmt.Sprintf("https://wordpress.org/plugins/%s/", plugin)
	var response []byte = httpRequest(urlStr)
	var output string = string(response)
	re := regexp.MustCompile(`(\d+) of (\d+) support threads .+ have been marked resolved`)

	var matches []string = re.FindAllString(output, -1)

	if len(matches) > 0 {
		fmt.Printf("\n%s\n", matches[0])
	}
}

func analyzePageTickets(wg *sync.WaitGroup, plugin string, page int) {
	var urlStr string = fmt.Sprintf("https://wordpress.org/support/plugin/%s/page/%d", plugin, page)
	var response []byte = httpRequest(urlStr)
	var output string = string(response)

	if strings.Contains(output, "bbp-topics") {
		var resolved int = strings.Count(output, ">[Resolved]")
		var resolvedWithPadding string = fmt.Sprintf("%2d", resolved)
		var pageWithPadding string = fmt.Sprintf("%2d", page)
		var maximumPerPage int = strings.Count(output, "<ul id=\"bbp-topic-")
		var status string

		if resolved == maximumPerPage {
			status = fmt.Sprintf("\033[0;92m%s\033[0m", "\u2714")
		} else {
			var missing int = maximumPerPage - resolved

			if missing > 6 {
				status = fmt.Sprintf("\033[0;91m%s\033[0m", "\u2718")
			} else if missing > 3 {
				status = fmt.Sprintf("\033[0;93m%s\033[0m", "\u2622")
			} else {
				status = fmt.Sprintf("\033[0;94m%s\033[0m", "\u2022")
			}

			status += fmt.Sprintf(" (%d missing) %s", missing, urlStr)
		}

		fmt.Printf("- Page %s %s/%d %s\n",
			pageWithPadding,
			resolvedWithPadding,
			maximumPerPage,
			status)
	}

	defer wg.Done()
}

func main() {
	flag.Parse()

	var plugin string = flag.Arg(0)
	var pages string = flag.Arg(1)
	var limit int = 10

	if plugin == "" {
		fmt.Println("WordPress Tickets")
		fmt.Println("  https://cixtor.com/")
		fmt.Println("  https://wordpress.org/support/")
		fmt.Println("  https://github.com/cixtor/wptickets")
		fmt.Println("Usage: wptickets [plugin] [pages]\n")
		os.Exit(2)
	}

	fmt.Printf("Plugin.: %s\n", plugin)
	fmt.Printf("Website: https://wordpress.org/plugins/%s/\n", plugin)
	fmt.Printf("Support: https://wordpress.org/support/plugin/%s/\n", plugin)
	fmt.Printf("\n")
	fmt.Printf("Resolved threads:\n")

	if pages != "" {
		number, err := strconv.Atoi(pages)

		if err == nil {
			limit = number
		}
	}

	var wg sync.WaitGroup

	wg.Add(limit)

	for key := 1; key <= limit; key++ {
		go analyzePageTickets(&wg, plugin, key)
	}

	wg.Wait()

	analyzeMonthStats(plugin)

	os.Exit(0)
}
