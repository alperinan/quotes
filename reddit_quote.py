#!/usr/bin/env python3
# coding: utf-8






# your function must import layout and constants
# this is structured to work both in Jupyter notebook and from the command line
try:
    from . import layout
    from . import constants
except ImportError:
    import layout
    import constants

import logging
import re
import json
import secrets
from time import time
from pathlib import Path
from datetime import datetime
from os import path

import requests
from dictor import dictor
import sys
from typing import List, Dict, Any, Optional
import sqlite3
import os






# fugly hack for making the library module available to the plugins
sys.path.append(layout.dir_path+'/../..')
from library import PluginTools






def _time_now():
    return datetime.now().strftime("%H:%M")






def _fetch_quotes():
    '''fetch quotes from reddit'''
    error = False
    logging.debug('fetching data from reddit')
    raw_quotes = [constants.error_text]
    try:
        r = requests.get(constants.quotes_url, headers=constants.headers)
    except requests.RequestException as e:
        logging.error(f'failed to fetch quotes from {constants.quotes_url}, {e}')
        return (raw_quotes, True)
    if r.status_code == 200:
        try:
            json_data = dictor(r.json(), constants.quote_data_addr)
            raw_quotes = [dictor(q, constants.quote_title_addr) for q in json_data]
        except json.JSONDecodeError as e:
            logging.error(f'bad json data: {e}')
            raw_quotes = [constants.error_text]
            error = True
    else:
        logging.warning(f'error accessing {constants.quotes_url}: code {r.status_code}')
        raw_quotes = [constants.error_text]
        error = True
        
    if len(raw_quotes) < 1:
        raw_quotes = [constants.error_text]
        error = True
        
    return (raw_quotes, error)






def _process_quotes(raw_quotes):
    processed_quotes = []
    logging.debug(f'processing {len(raw_quotes)} quotes')
    for quote in raw_quotes:
        # make sure we have a string to work with
        quote = str(quote)
        # sub double quotes for any other quote character or '' 
        q = re.sub('“|”|\'\'|"', '', quote)
        # sub single quote for ’ character
        q = re.sub('’', "'", q)
        # sub minus for endash, emdash, hyphen, ~
        q = re.sub('-|–|—|~|--|―', '-', q)
        # clean trailing whitespace in quotes
        q = re.sub('\s+"', '"', q)
        # split quote from attirbution
        match = re.match('(.*)\s{0,}-\s{0,}(.*)', q)

        if hasattr(match, 'groups'):
            if len(match.groups()) > 1:
                text = match.group(1).strip()
                attribution = match.group(2).strip().title()
            else:
                text = match.group(1).strip()
                attribution = None
        else:
            text = q.strip()
            attribution = None

        # append quotes to dictionary
        
        processed_quotes.append({'len': len(q), 'text': text, 'attribution': attribution})
    return processed_quotes


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

def get_random_quote(conn: sqlite3.Connection) -> Optional[Dict[str, Any]]:
    """
    Select one row from quotes table preferring rows with smaller viewCount but
    randomizing among ties, increment its viewCount, and return it as a dict.
    """
    #processed_quotes = []

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

    #processed_quotes.append({'len': len(text), 'text': text, 'attribution': author})

    return {
        "len": len(text),
        "text": text,
        "attribution": author
    }

# make sure this function can accept *args and **kwargs even if you don't intend to use them
def update_function(self, *args, **kwargs):
    '''update function for reddit_quote plugin
    
    Scrapes quotes from reddit.com/r/quotes and displays them one at a time
    
   Requirements:
        self.config(`dict`): {
        'max_length': 144,   # name of player to track
        'idle_timeout': 10,               # timeout for disabling plugin
    }
    self.cache(`CacheFiles` object)

    Args:
        self(namespace): namespace from plugin object
        
    Returns:
        tuple: (is_updated(bool), data(dict), priority(int))   
        
    This plugin is inspired by and based on the veeb.ch [stonks project](https://github.com/veebch/stonks)
    
    %U'''  

    
    logging.info(f'update function for {constants.name}')
    json_file = self.cache.path/Path(constants.json_file)
    
    max_length = self.config.get('max_length', constants.required_config_options['max_length'])
    max_retries = self.config.get('max_retries', constants.required_config_options['max_retries'])
    
    try:
        max_length = int(max_length)
        max_retries = int(max_retries)
    except ValueError as e:
        logging.warning('non-numeric values provided in configuration file for max_length or max_retries')
    
    is_updated = False
    data = {}
    priority = 2**16
    
    conn = create_or_connect_db()

    my_quote = get_random_quote(conn)
    if not my_quote:
        print("No quotes found in the database.")
        return
    author = my_quote["author"] or "Unknown"                    

    #my_quote['len']=''
    #my_quote['attribution'] = ''
    #my_quote['text'] = ''

    if my_quote['attribution']:
        attribution = my_quote['attribution']
        my_quote['attribution'] = f'{constants.attribution_char}{attribution}'

    data = my_quote
    data['time'] = _time_now()
    data['tag_image'] = constants.tag_image
    is_updated = True
    priority = self.max_priority

    if 'text_color' in self.config or 'bkground_color' in self.config:
        logging.info('using user-defined colors')
        colors = PluginTools.text_color(config=self.config, mode=self.screen_mode,
                               default_text=self.layout.get('fill', 'WHITE'),
                               default_bkground=self.layout.get('bkground', 'BLACK'))

        text_color = colors['text_color']
        bkground_color = colors['bkground_color']


        # set the colors
        logging.debug(f'trying to set fill and background for sections: {list(self.layout.keys())}')
        for section in self.layout:
            if self.layout[section].get('rgb_support', False):
                logging.debug(f'setting {section} layout colors to fill: {text_color}, bkground: {bkground_color}')
                self.layout_obj.update_block_props(section, {'fill': text_color, 'bkground': bkground_color}) 

            else:
                logging.debug(f'section {section} does not support RGB colors')
        
        
    return (is_updated, data, priority)








# # this code snip simulates running from within the display loop use this and the following
# # cell to test the output
# import logging
# logging.root.setLevel('DEBUG')
# from library.CacheFiles import CacheFiles
# from library.Plugin import Plugin
# from IPython.display import display

# test_plugin = Plugin(resolution=(800, 600), screen_mode='RGB')
# test_plugin.refresh_rate = 5
# l = layout.layout
# # l = layout.quote
# test_plugin.config = {
#     'text_color': 'RED',
#     'bkground_color': 'random'
# }
# test_plugin.layout = l
# test_plugin.cache = CacheFiles()
# test_plugin.update_function = update_function
# test_plugin.update()
# test_plugin.image









