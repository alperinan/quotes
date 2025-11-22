import os
try:
    import requests
except Exception:
    requests = None
from urllib.request import Request, urlopen
import re
import html
import json
import time

def download_and_save_html(url="https://1000kitap.com/konu/alinti?sayfa=2", filename="webfile2.txt"):
    """
    Download HTML from `url` and save it next to this script as `filename`.
    Uses requests if installed, otherwise falls back to urllib.
    Returns saved file path.
    """
    folder = os.path.dirname(os.path.abspath(__file__))
    out_path = os.path.join(folder, filename)
    headers = {"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"}

    if requests:
        resp = requests.get(url, headers=headers, timeout=15)
        resp.raise_for_status()
        encoding = resp.encoding or "utf-8"
        text = resp.text
    else:
        req = Request(url, headers=headers)
        with urlopen(req, timeout=15) as r:
            raw = r.read()
            info = r.info()
            encoding = None
            if hasattr(info, "get_content_charset"):
                encoding = info.get_content_charset()
            encoding = encoding or "utf-8"
            text = raw.decode(encoding, errors="replace")

    with open(out_path, "w", encoding="utf-8") as f:
        f.write(text)
    return out_path

def parse_quotes_from_file(filename="webfile.txt"):
    """
    Parse saved HTML file and return a list of quote dicts:
      [{ "quote": "...", "author": "...", "book": "..." }, ...]
    This version targets spans with class containing 'text-15' (e.g. <span class="text text text-15">)
    and then tries to find nearby attribution (author/book) in the surrounding HTML.
    """
    folder = os.path.dirname(os.path.abspath(__file__))
    path = os.path.join(folder, filename)

    try:
        with open(path, "r", encoding="utf-8") as fh:
            html_text = fh.read()
    except FileNotFoundError:
        return []

    def strip_tags(s):
        s = re.sub(r'<script.*?>.*?</script>', '', s, flags=re.DOTALL | re.IGNORECASE)
        s = re.sub(r'<style.*?>.*?</style>', '', s, flags=re.DOTALL | re.IGNORECASE)
        s = re.sub(r'<[^>]+>', '', s)
        s = html.unescape(s)
        s = re.sub(r'\s+\n', '\n', s)
        s = re.sub(r'\n\s+', '\n', s)
        s = re.sub(r'[ \t]+', ' ', s).strip()
        return s

    results = []
    seen = set()

    # find all spans whose class contains text-15 (matches "text text text-15" etc.)
    span_re = re.compile(r'<span\b[^>]*class=[\'"][^\'"]*\btext-15\b[^\'"]*[\'"][^>]*>(.*?)</span>',
                         re.IGNORECASE | re.DOTALL)
    for m in span_re.finditer(html_text):
        inner_html = m.group(1)
        quote_text = strip_tags(inner_html)
        if not quote_text:
            continue

        # look in a nearby window after the span for attribution (author/book)
        window_after = strip_tags(html_text[m.end(): m.end() + 800])
        window_before = strip_tags(html_text[max(0, m.start() - 400): m.start()])

        author = ""
        book = ""

        # Try 1: em-dash or dash in the after-window: "— Author, Book" or "- Author (Book)"
        m1 = re.search(r'^[\s\-\u2013\u2014—]{0,4}\s*([^,\(\n]{2,80})(?:[,\s]*\(?\s*([^)\n]{2,120})\s*\)?)?',
                       window_after.strip(), re.MULTILINE)
        if m1 and m1.group(1):
            author = m1.group(1).strip()
            if m1.lastindex >= 2 and m1.group(2):
                book = m1.group(2).strip()

        # Try 2: look for typical author tags near the span (e.g. <a class="author">Name</a> or <span class="author">)
        if not author:
            nearby = html_text[max(0, m.start()-300): m.end()+300]
            m2 = re.search(r'<(?:a|span|div)\b[^>]*class=[\'"][^\'"]*(author|yazar|writer|user|name|meta)[^\'"]*[\'"][^>]*>(.*?)</(?:a|span|div)>',
                           nearby, re.IGNORECASE | re.DOTALL)
            if m2:
                author = strip_tags(m2.group(2))

        # Try 3: parenthesized attribution immediately after the quote in the after-window
        if not author:
            m3 = re.search(r'^\s*[—\-\u2013\u2014]?\s*(.+?)\s*(?:\(|,|$)', window_after.strip(), re.MULTILINE)
            if m3:
                cand = m3.group(1).strip()
                if 2 <= len(cand) <= 80:
                    author = cand

        # Fallback heuristics using combined text (quote + before/after) to catch patterns like "Quote — Author"
        if not author:
            combined = (quote_text + "\n" + window_after).strip()
            m4 = re.search(r'(.+?)[\s\u2013\u2014\-]{1,3}\s*([^,(\n]{2,80})(?:[,(]\s*(.+?)\s*[)\n])?$', combined, re.DOTALL)
            if m4:
                # ensure the left part starts with the extracted quote or is very similar
                left = m4.group(1).strip()
                if quote_text.startswith(left) or left.startswith(quote_text) or len(left) < len(quote_text) + 30:
                    author = m4.group(2).strip()
                    if m4.lastindex >= 3 and m4.group(3):
                        book = m4.group(3).strip()

        # cleanup
        quote = quote_text.strip(' "\'«»“”\n\r\t ')
        author = author.strip(' "\'«»“”\n\r\t ')
        book = book.strip(' "\'«»“”\n\r\t ')

        key = (quote, author, book)
        if quote and key not in seen:
            seen.add(key)
            results.append({"quote": quote, "author": author, "book": book})

    return results

if __name__ == "__main__":

    for i in range(73,101):
        baseUrl = "https://1000kitap.com/konu/alinti?sayfa="
        baseFileName = "webfile"
        download_and_save_html(baseUrl + str(i+1), baseFileName + str(i+1) + ".txt")
        time.sleep(30)
    
    
    #quotes = parse_quotes_from_file()
    #out_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "quotes.json")
    #with open(out_path, "w", encoding="utf-8") as fh:
    #    json.dump(quotes, fh, ensure_ascii=False, indent=2)