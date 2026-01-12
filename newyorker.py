#!/usr/bin/env python3
# coding: utf-8








import logging






from random import randrange
from datetime import datetime
from pathlib import Path






# two different import modes for development or distribution
try:
    # import from other modules above this level
    from . import layout
    from . import constants
except ImportError:
    import constants
    # development in jupyter notebook
    import layout




from typing import List, Dict, Any, Optional


import feedparser
import requests

import sqlite3
import os




logger = logging.getLogger(__name__)



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
        CREATE TABLE IF NOT EXISTS funFacts (
            id TEXT PRIMARY KEY,
            text TEXT NOT NULL,
            viewCount INTEGER DEFAULT 0
        )
        """
    )
    conn.commit()
    return conn

def get_random_fact(conn: sqlite3.Connection) -> Optional[Dict[str, Any]]:
    import random
    
    cur = conn.cursor()
    cur.execute(
        """
        SELECT id, text, COALESCE(viewCount, 0)
        FROM funFacts
        ORDER BY COALESCE(viewCount, 0) ASC, RANDOM()
        LIMIT 1
        """
    )
    row = cur.fetchone()
    if not row:
        return None

    qid, text, vc = row[0], row[1], row[2] or 0
    # increment viewCount for the selected row
    cur.execute("UPDATE funFacts SET viewCount = COALESCE(viewCount, 0) + 1 WHERE id = ?", (qid,))
    conn.commit()

    # Generate random number between 1 and 12
    random_number = random.randint(1, 12)
    imgName = f"funFact{random_number}.jpg"

    return {
        "len": len(text),
        "text": text,
        "attribution": imgName
    }




def update_function(self, **kwargs):
    '''update function for newyorker provides a New Yorker comic of the day
    
    This plugin provides an image and text pulled from the New Yorker 
    
    Requirments:
        self.config(dict): {
            'day_range': 'number of days to pull comics from (default: 5)',
        }    
    
    Args:
        self(`namespace`)
        day_range(`int`): number of days in the past to pull radom comic and text from
            use 1 to only pull from today
        
    Returns:
        tuple: (is_updated(bool), data(dict), priority(int))
    
    This plugin is inspired and based on the veeb.ch [stonks project](https://github.com/veebch/stonks)
        
    %U'''
    def time_now():
        return datetime.now().strftime("%H:%M")


    conn = create_or_connect_db()

    my_fact = get_random_fact(conn)

    is_updated = True
    
    data = {'comic': Path(constants.images_path)/my_fact['attribution'],
            'caption': my_fact['text'],
            'time': time_now()
            }

    priority = self.max_priority
    
    return (is_updated, data, priority)












