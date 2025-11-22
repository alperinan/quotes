import os
import json
from typing import List, Dict, Any, Optional
import sqlite3

def parse_quotes_json(filename: str | None = None) -> List[Dict]:
    """
    Read quotes.json from the same folder as this file and return a list of quote objects.
    """
    if filename is None:
        filename = os.path.join(os.path.dirname(__file__), "quotes.json")
    try:
        with open(filename, "r", encoding="utf-8") as f:
            data = json.load(f)
        if isinstance(data, list):
            return data
        if isinstance(data, dict):
            # common pattern: {"quotes": [...]}
            for key in ("quotes", "data", "items"):
                if key in data and isinstance(data[key], list):
                    return data[key]
            return [data]
        return []
    except FileNotFoundError:
        print(f"quotes.json not found at {filename}")
        return []
    except json.JSONDecodeError as e:
        print(f"Error parsing JSON: {e}")
        return []

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

def insert_quotes(conn: sqlite3.Connection, quotes: List[Dict], lang: str = "en") -> int:
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

def get_random_quote(conn: sqlite3.Connection) -> Optional[Dict[str, Any]]:
    """
    Select one row from quotes table preferring rows with smaller viewCount but
    randomizing among ties, increment its viewCount, and return it as a dict.
    """
    cur = conn.cursor()
    cur.execute(
        """
        SELECT id, text, author, lang, COALESCE(viewCount, 0)
        FROM quotes
        ORDER BY COALESCE(viewCount, 0) ASC, RANDOM()
        LIMIT 1
        """
    )
    row = cur.fetchone()
    if not row:
        return None

    qid, text, author, lang, vc = row[0], row[1], row[2], row[3], row[4] or 0
    # increment viewCount for the selected row
    cur.execute("UPDATE quotes SET viewCount = COALESCE(viewCount, 0) + 1 WHERE id = ?", (qid,))
    conn.commit()

    return {
        "id": qid,
        "text": text,
        "author": author,
        "lang": lang,
        "viewCount": vc + 1,
    }

def print_random_quote(conn: sqlite3.Connection) -> None:
    """
    Fetch a random quote and print it to the screen.
    """
    q = get_random_quote(conn)
    if not q:
        print("No quotes found in the database.")
        return
    author = q["author"] or "Unknown"
    print(f'"{q["text"]}" — {author} (lang={q["lang"]}, id={q["id"]}, views={q["viewCount"]})')

if __name__ == "__main__":
    quotes = parse_quotes_json()
    print(f"Loaded {len(quotes)} quotes")
    #if quotes:
    #    print(quotes[0])
    conn = create_or_connect_db()
    #print(f"Connected to database at {os.path.join(os.path.dirname(__file__), 'database.db')}")
    #inserted = insert_quotes(conn, quotes, lang="en")
    #print(f"Inserted {inserted} quotes into the database")
    print_random_quote(conn)
    conn.close()