package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Quote represents a single quote with its metadata
type Quote struct {
	QuoteText string `json:"quoteText"`
	Author    string `json:"author"`
	BookName  string `json:"bookName"`
	BookLink  string `json:"bookLink"`
}

// parse1000KitapQuotes parses HTML content and extracts quotes as an array of Quote structs
func parse1000KitapQuotes(htmlContent string) ([]Quote, error) {
	var quotes []Quote
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	bookHrefRe := regexp.MustCompile(`^/kitap/([^/]+)--(\d+)`)
	authorHrefRe := regexp.MustCompile(`^/yazar/([^/]+)`)

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			class := getAttr(n, "class")
			if class == "text text text-15" {
				quoteText := textOfNode(n)
				parent := n.Parent
				var author, bookName, bookLink string
				if parent != nil {
					for c := parent.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.ElementNode && c.Data == "a" {
							href := getAttr(c, "href")
							title := textOfNode(c)
							switch {
							case bookHrefRe.MatchString(href):
								bookName = title
								bookLink = "https://1000kitap.com" + href
							case authorHrefRe.MatchString(href):
								author = title
							}
						}
					}
				}
				// Clean up fields for SQLite
				quoteText = sanitizeForSQLite(quoteText)
				author = sanitizeForSQLite(author)
				bookName = sanitizeForSQLite(bookName)
				if quoteText != "" && author != "" && bookName != "" && bookLink != "" {
					quotes = append(quotes, Quote{
						QuoteText: quoteText,
						Author:    author,
						BookName:  bookName,
						BookLink:  bookLink,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// Try also to parse __NEXT_DATA__ <script> tag for fallback
	if len(quotes) == 0 {
		start := strings.Index(htmlContent, `id="__NEXT_DATA__"`)
		if start > 0 {
			scriptTag := htmlContent[start:]
			startJSON := strings.Index(scriptTag, ">") + 1
			endJSON := strings.Index(scriptTag, "</script>")
			if startJSON > 0 && endJSON > startJSON {
				jsonStr := scriptTag[startJSON:endJSON]
				var nextData map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &nextData); err == nil {
					// Traverse into pageProps/response/_sonuc/gonderiler
					props := getMap(nextData, "props")
					pageProps := getMap(props, "pageProps")
					resp := getMap(pageProps, "response")
					_sonuc := getMap(resp, "_sonuc")
					gonderiler, ok := _sonuc["gonderiler"].([]interface{})
					if ok {
						for _, item := range gonderiler {
							post, ok := item.(map[string]interface{})
							if !ok {
								continue
							}
							if turu, _ := post["turu"].(string); turu == "sozler" {
								alt := getMap(post, "alt")
								kitaplar := getMap(alt, "kitaplar")
								yazarlar := getMap(alt, "yazarlar")
								sozler := getMap(alt, "sozler")
								sozParse := getMap(sozler, "sozParse")
								parse := sozParse["parse"]
								var quoteText string
								switch v := parse.(type) {
								case []interface{}:
									var b strings.Builder
									for _, s := range v {
										if sstr, ok := s.(string); ok {
											b.WriteString(sstr)
										}
									}
									quoteText = b.String()
								case string:
									quoteText = v
								}
								bookName, _ := kitaplar["adi"].(string)
								bookID, _ := kitaplar["id"].(string)
								bookSlug, _ := kitaplar["seo_adi"].(string)
								authorName, _ := yazarlar["adi"].(string)
								bookLink := fmt.Sprintf("https://1000kitap.com/kitap/%s--%s", bookSlug, bookID)
								// Clean up fields for SQLite
								quoteText = sanitizeForSQLite(quoteText)
								authorName = sanitizeForSQLite(authorName)
								bookName = sanitizeForSQLite(bookName)
								if quoteText != "" && authorName != "" && bookName != "" && bookLink != "" {
									quotes = append(quotes, Quote{
										QuoteText: quoteText,
										Author:    authorName,
										BookName:  bookName,
										BookLink:  bookLink,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	return quotes, nil
}

// Helper function: get attribute value by name
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// Helper function: strip HTML tags from a string
func stripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}

// Helper function: get text content of node, cleaned from HTML tags
func cleanText(s string) string {
	return stripHTMLTags(html.UnescapeString(s))
}

// Helper function: get text content of node
func textOfNode(n *html.Node) string {
	var b strings.Builder
	var f func(*html.Node)
	f = func(nd *html.Node) {
		if nd.Type == html.TextNode {
			b.WriteString(nd.Data)
		}
		for c := nd.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return cleanText(b.String())
}

// Helper function: get map[string]interface{} field
func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if raw, ok := m[key]; ok {
		if out, ok := raw.(map[string]interface{}); ok {
			return out
		}
	}
	return nil
}

// Helper function: clean up text for safe SQLite/JSON insertion
func sanitizeForSQLite(s string) string {
	// Remove HTML tags
	s = stripHTMLTags(html.UnescapeString(s))
	// Remove leading/trailing quotes (both straight and curly)
	s = strings.Trim(s, "\"\u2018\u2019\u201C\u201D'«»")
	// Replace problematic quotes and backslashes
	s = strings.ReplaceAll(s, "'", "'")
	s = strings.ReplaceAll(s, `"`, "")
	s = strings.ReplaceAll(s, `\`, "")
	// Remove newlines, tabs, carriage returns
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	// Remove control characters
	re := regexp.MustCompile(`[\x00-\x1F\x7F]+`)
	s = re.ReplaceAllString(s, "")
	// Collapse multiple spaces into one
	re2 := regexp.MustCompile(`\s+`)
	s = re2.ReplaceAllString(s, " ")
	// Trim spaces
	return strings.TrimSpace(s)
}

// Reads from a file and parses the quotes
func parse1000KitapQuotesFromFile(filename string) ([]Quote, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return parse1000KitapQuotes(string(b))
}

func main() {
	// Find all files starting with "webfile" in current folder
	files, err := filepath.Glob("webfile*.txt")
	if err != nil {
		log.Fatalf("Error finding files: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("No webfile*.txt files found in folder")
	}

	var allQuotes []Quote
	for _, filename := range files {
		quotes, err := parse1000KitapQuotesFromFile(filename)
		if err != nil {
			log.Printf("Error parsing %s: %v", filename, err)
			continue
		}
		allQuotes = append(allQuotes, quotes...)
	}

	// Save all quotes to quotes.json
	outFile := "quotes.json"
	fh, err := os.Create(outFile)
	if err != nil {
		log.Fatalf("Error creating %s: %v", outFile, err)
	}
	defer fh.Close()
	enc := json.NewEncoder(fh)
	enc.SetIndent("", "  ")
	if err := enc.Encode(allQuotes); err != nil {
		log.Fatalf("Error writing JSON: %v", err)
	}
	fmt.Printf("Parsed %d quotes from %d files. Saved to %s\n", len(allQuotes), len(files), outFile)
}
