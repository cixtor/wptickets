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
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func httpRequest(target string) io.Reader {
	req, err := http.NewRequest(http.MethodGet, target, nil)

	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("pragma", "no-cache")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("authority", "wordpress.org")
	req.Header.Set("accept-language", "en-US,en")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (KHTML, like Gecko) Safari/537.36")
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	(&buf).ReadFrom(resp.Body)

	return &buf
}

func analyzeMonthStats(plugin string) {
	urlStr := fmt.Sprintf("https://wordpress.org/plugins/%s/", plugin)
	response := httpRequest(urlStr)
	scanner := bufio.NewScanner(response)

	var line string

	for scanner.Scan() {
		line = scanner.Text()

		if strings.Contains(line, "\x20out of\x20") {
			line = strings.Replace(line, "</span>", "", -1)
			line = strings.TrimSpace(line)
			if line[0:4] == "<div" {
				continue
			}
			fmt.Println("\nIssues resolved in last two months: " + line)
			break
		}
	}
}

func analyzePageTickets(result chan string, plugin string, page int) {
	urlStr := fmt.Sprintf("https://wordpress.org/support/plugin/%s/page/%d", plugin, page)
	response := httpRequest(urlStr)
	scanner := bufio.NewScanner(response)
	pagepad := fmt.Sprintf("%2d", page)

	var resolvedpad string
	var status string
	var resolved int
	var line string
	var maximum int

	for scanner.Scan() {
		line = scanner.Text()

		if strings.Contains(line, "<ul id=\"bbp-topic-") {
			maximum++
		}

		if strings.Contains(line, "aria-label=\"Resolved\"") {
			resolved++
		}
	}

	if maximum == 0 {
		result <- ""
		return /* Non-existent page */
	}

	resolvedpad = fmt.Sprintf("%2d", resolved)

	if resolved == maximum {
		status = fmt.Sprintf("\033[0;92m%s\033[0m", "\u2714")
	} else {
		missing := maximum - resolved

		if missing > 6 {
			status = fmt.Sprintf("\033[0;91m%s\033[0m", "\u2718")
		} else if missing > 3 {
			status = fmt.Sprintf("\033[0;93m%s\033[0m", "\u2622")
		} else {
			status = fmt.Sprintf("\033[0;94m%s\033[0m", "\u2022")
		}

		status += fmt.Sprintf(" (%d missing) %s", missing, urlStr)
	}

	result <- fmt.Sprintf("- Page %s %s/%d %s",
		pagepad,
		resolvedpad,
		maximum,
		status)
}

func reportResults(results []string) {
	sort.Strings(results)

	for _, stats := range results {
		if stats != "" {
			fmt.Println(stats)
		}
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "WordPress Tickets\n")
		fmt.Fprintf(os.Stderr, "https://cixtor.com/\n")
		fmt.Fprintf(os.Stderr, "https://wordpress.org/support/\n")
		fmt.Fprintf(os.Stderr, "https://github.com/cixtor/wptickets\n")
		fmt.Fprintf(os.Stderr, "\nUsage:\n")
		fmt.Fprintf(os.Stderr, "  wptickets [plugin]\n")
		fmt.Fprintf(os.Stderr, "  wptickets [plugin] [pages]\n")

		flag.PrintDefaults()
	}

	flag.Parse()

	plugin := flag.Arg(0)
	pages := flag.Arg(1)
	limit := 10

	if plugin == "" {
		flag.Usage()
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

	var final []string

	result := make(chan string)

	for key := 1; key <= limit; key++ {
		go analyzePageTickets(result, plugin, key)
	}

	for i := 0; i < limit; i++ {
		data := <-result
		final = append(final, data)
	}

	reportResults(final)
	analyzeMonthStats(plugin)
}
