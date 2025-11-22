import re
from bs4 import BeautifulSoup

def parse_1000kitap_quotes(html_content):
    """
    Parses an HTML from 1000kitap quote pages and returns an array of quotes.
    Each quote dictionary contains quoteText, author, bookName, and bookLink.
    """
    soup = BeautifulSoup(html_content, 'html.parser')
    result = []

    # Look for quote blocks by possible classes or text indicators.
    # Each quote is in a sequence of elements that must include quote, author, book name, and book link.
    # We'll use robust regex and soup queries to search.

    # Find all quote blocks: look for <div> containing 'Kitabı okudu' or 'Kitabı okuyor' and then its parent.
    for span in soup.find_all('span', class_='text text text-15'):
        text = span.get_text(strip=True)
        quote_text = text.replace('“', '').replace('”', '').strip()

        # Traverse up to get related book and author info
        quote_parent = span.find_parent('div', class_='dr')
        author = None
        book_name = None
        book_link = None

        # Look for <a> tags to author and book within the same parent
        for a in quote_parent.find_all('a'):
            href = a.get('href', '')
            atext = a.get_text(strip=True)
            # Book links look like: /kitap/<slug>
            if href.startswith('/kitap/'):
                book_name = atext
                book_link = 'https://1000kitap.com' + href
            # Author links look like: /yazar/<slug>
            elif href.startswith('/yazar/'):
                author = atext

        if quote_text and author and book_name and book_link:
            result.append({
                'quoteText': quote_text,
                'author': author,
                'bookName': book_name,
                'bookLink': book_link
            })

    # Fallback: look for generic quote sections in the API data
    # If quote blocks missed above, try to parse raw text as fallback
    script = soup.find('script', id='__NEXT_DATA__', type='application/json')
    if script:
        import json
        data = json.loads(script.text)
        try:
            posts = data['props']['pageProps']['response']['_sonuc']['gonderiler']
            for post in posts:
                if post.get('turu') == 'sozler':
                    alt = post['alt']['sozler']
                    textList = alt['sozParse']['parse']
                    # Some are lists, some are just a string
                    quote_text = ''
                    if isinstance(textList, list):
                        quote_text = ''.join(str(part) if isinstance(part, str) else '' for part in textList).replace('“', '').replace('”', '').strip()
                    else:
                        quote_text = textList
                    yazar = post['alt']['yazarlar']['adi']
                    kitap = post['alt']['kitaplar']['adi']
                    kitap_id = post['alt']['kitaplar']['id']
                    # Compose book link
                    kitap_slug = post['alt']['kitaplar']['seo_adi']
                    book_link = f'https://1000kitap.com/kitap/{kitap_slug}--{kitap_id}'
                    result.append({
                        'quoteText': quote_text,
                        'author': yazar,
                        'bookName': kitap,
                        'bookLink': book_link
                    })
        except Exception:
            pass

    return result

# Example usage:
# with open('webfile3.txt', 'r', encoding='utf-8') as f:
#     html_content = f.read()
# quotes = parse_1000kitap_quotes(html_content)
# print(quotes)