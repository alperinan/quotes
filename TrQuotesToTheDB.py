import os
import json
from typing import List, Dict, Any, Optional
import sqlite3

def parse_quotes_json(filename: str | None = None) -> List[Dict]:
    """
    Read quotes.json from the same folder as this file and return a list of quote objects.
    Each object has 'text' and 'author' keys suitable for insert_quotes method.
    """
    if filename is None:
        filename = os.path.join(os.path.dirname(__file__), "quotes.json")
    
    if not os.path.exists(filename):
        print(f"File not found: {filename}")
        return []
    
    try:
        with open(filename, "r", encoding="utf-8") as f:
            data = json.load(f)
    except Exception as e:
        print(f"Error reading JSON file: {e}")
        return []
    
    quotes = []
    for item in data:
        if not isinstance(item, dict):
            continue
        
        text = item.get("quoteText", "").strip()
        author = item.get("author", "").strip()
        book_name = item.get("bookName", "").strip()
        
        if not text:
            continue
        
        # Combine author and book name with '-'
        author_field = ""
        if author and book_name:
            author_field = f"{author} - {book_name}"
        elif author:
            author_field = author
        elif book_name:
            author_field = book_name
        
        quotes.append({
            "text": text,
            "author": author_field if author_field else None
        })
    
    return quotes

def create_or_connect_db(db_path: str | None = None) -> sqlite3.Connection:
    """
    Create or connect to a SQLite database file (default: database.db in this folder)
    and ensure a 'quotes' table exists with columns: id, text, author, lang.
    Returns the sqlite3.Connection object.
    """
    if db_path is None:
        db_path = os.path.join(os.path.dirname(__file__), "database.db")
    conn = sqlite3.connect(db_path)
    cur = conn.cursor()
    cur.execute(
        """
        CREATE TABLE IF NOT EXISTS quotes (
            id INTEGER PRIMARY KEY,
            text TEXT NOT NULL,
            author TEXT,
            lang TEXT,
            viewCount INTEGER DEFAULT 0
        )
        """
    )
    conn.commit()
    return conn

def insert_quotes(conn: sqlite3.Connection, quotes: List[Dict], lang: str = "tr") -> int:
    """
    Insert a list of quote objects into the 'quotes' table.
    Uses 'en' for the lang column by default.
    Returns the number of rows inserted.
    """
    rows = []
    for q in quotes:
        if isinstance(q, dict):
            text = q.get("text") or q.get("quote") or q.get("body")
            author = q.get("author") or q.get("by") or q.get("source")
        else:
            # if quote is a plain string or other type, convert to str and leave author None
            text = str(q)
            author = None
        if not text:
            continue
        rows.append((text, author, lang))
    if not rows:
        return 0
    cur = conn.cursor()
    cur.executemany(
        "INSERT INTO quotes (text, author, lang) VALUES (?, ?, ?)",
        rows
    )
    conn.commit()
    return cur.rowcount



if __name__ == "__main__":
    quotes = parse_quotes_json()
    
    print(f"Loaded {len(quotes)} quotes\n")

    for idx, quote in enumerate(quotes, 1):
        print(f"Quote #{idx}:")
        print(f"  Text: {quote.get('text', '')}")
        print(f"  Author: {quote.get('author', '')}")
        print()


    conn = create_or_connect_db()

    inserted = insert_quotes(conn, quotes, lang="tr")
    print(f"Inserted {inserted} quotes into the database")

    conn.close()