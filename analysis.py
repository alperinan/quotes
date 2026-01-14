import sqlite3
import difflib
from collections import defaultdict
from typing import List, Tuple, Dict
import sys

def connect_db(db_path: str = "database.db") -> sqlite3.Connection:
    """Connect to the SQLite database"""
    return sqlite3.connect(db_path)

def get_all_quotes(conn: sqlite3.Connection) -> List[Tuple[int, str, str]]:
    """Retrieve all quotes with id, text, and author"""
    cursor = conn.cursor()
    cursor.execute("SELECT id, text, author FROM quotes")
    return cursor.fetchall()

def normalize_text(text: str) -> str:
    """Normalize text for comparison (lowercase, strip whitespace)"""
    return text.lower().strip()

def find_similar_quotes(quotes: List[Tuple[int, str, str]], similarity_threshold: float = 0.85) -> List[Dict]:
    """
    Find similar quotes based on text + author combination
    Returns list of similar quote groups
    """
    similar_groups = []
    processed = set()
    total = len(quotes)
    
    print(f"\n   Processing {total} quotes for similarity...")
    
    for i, (id1, text1, author1) in enumerate(quotes):
        # Progress indicator
        if i % 100 == 0 or i == total - 1:
            progress = (i + 1) / total * 100
            sys.stdout.write(f"\r   Progress: {i + 1}/{total} ({progress:.1f}%) - Found {len(similar_groups)} groups")
            sys.stdout.flush()
        
        if id1 in processed:
            continue
            
        # Combine text and author for comparison
        combined1 = f"{normalize_text(text1)} | {normalize_text(author1)}"
        
        similar_group = {
            'main': {'id': id1, 'text': text1, 'author': author1},
            'similar': []
        }
        
        for j, (id2, text2, author2) in enumerate(quotes):
            if i >= j or id2 in processed:
                continue
            
            combined2 = f"{normalize_text(text2)} | {normalize_text(author2)}"
            
            # Calculate similarity ratio
            similarity = difflib.SequenceMatcher(None, combined1, combined2).ratio()
            
            if similarity >= similarity_threshold:
                similar_group['similar'].append({
                    'id': id2,
                    'text': text2,
                    'author': author2,
                    'similarity': similarity
                })
                processed.add(id2)
        
        # Only add groups with similar quotes
        if similar_group['similar']:
            processed.add(id1)
            similar_groups.append(similar_group)
    
    print()  # New line after progress
    return similar_groups

def find_exact_duplicates(quotes: List[Tuple[int, str, str]]) -> Dict[str, List[Dict]]:
    """Find exact duplicates based on normalized text + author"""
    duplicates = defaultdict(list)
    total = len(quotes)
    
    print(f"\n   Processing {total} quotes for exact duplicates...")
    
    for i, (id, text, author) in enumerate(quotes):
        # Progress indicator
        if i % 100 == 0 or i == total - 1:
            progress = (i + 1) / total * 100
            sys.stdout.write(f"\r   Progress: {i + 1}/{total} ({progress:.1f}%)")
            sys.stdout.flush()
        
        key = f"{normalize_text(text)} | {normalize_text(author)}"
        duplicates[key].append({'id': id, 'text': text, 'author': author})
    
    print()  # New line after progress
    
    # Filter out keys with only one quote
    return {k: v for k, v in duplicates.items() if len(v) > 1}

def analyze_by_author(quotes: List[Tuple[int, str, str]]) -> Dict[str, int]:
    """Count quotes per author"""
    author_counts = defaultdict(int)
    for _, _, author in quotes:
        author_counts[author] += 1
    return dict(sorted(author_counts.items(), key=lambda x: x[1], reverse=True))

def delete_duplicates_and_similar(conn: sqlite3.Connection, dry_run: bool = True):
    """
    Delete exact duplicates and similar quotes from the database
    If dry_run=True, only shows what would be deleted without actually deleting
    """
    quotes = get_all_quotes(conn)
    ids_to_delete = set()
    
    print("\n" + "=" * 80)
    print("FINDING DUPLICATES AND SIMILAR QUOTES TO DELETE")
    print("=" * 80)
    
    # 1. Find exact duplicates (keep first occurrence, delete rest)
    print("\n1. Processing exact duplicates...")
    exact_dupes = find_exact_duplicates(quotes)
    
    if exact_dupes:
        print(f"   Found {len(exact_dupes)} groups of exact duplicates")
        for key, group in exact_dupes.items():
            # Keep the first ID, mark rest for deletion
            for quote in group[1:]:
                ids_to_delete.add(quote['id'])
                if len(ids_to_delete) % 100 == 0:
                    sys.stdout.write(f"\r   Marked {len(ids_to_delete)} quotes for deletion...")
                    sys.stdout.flush()
        print(f"\r   Marked {len(ids_to_delete)} quotes for deletion (exact duplicates)")
    else:
        print("   No exact duplicates found")
    
    # 2. Find similar quotes (keep first occurrence, delete rest)
    print("\n2. Processing similar quotes (85%+ similarity)...")
    similar_groups = find_similar_quotes(quotes, similarity_threshold=0.85)
    
    initial_count = len(ids_to_delete)
    if similar_groups:
        print(f"   Found {len(similar_groups)} groups of similar quotes")
        for i, group in enumerate(similar_groups):
            if i % 10 == 0:
                sys.stdout.write(f"\r   Processing group {i + 1}/{len(similar_groups)}...")
                sys.stdout.flush()
            # Mark all similar quotes for deletion (keep main)
            for sim in group['similar']:
                ids_to_delete.add(sim['id'])
        new_count = len(ids_to_delete) - initial_count
        print(f"\r   Marked {new_count} additional quotes for deletion (similar)")
    else:
        print("   No similar quotes found")
    
    # Summary
    print("\n" + "=" * 80)
    print(f"SUMMARY: {len(ids_to_delete)} quotes marked for deletion")
    print("=" * 80)
    
    if not ids_to_delete:
        print("\nNo quotes to delete. Database is clean!")
        return 0
    
    # Delete or show what would be deleted
    if dry_run:
        print("\n⚠️  DRY RUN MODE - No quotes will be deleted")
        print("Run with dry_run=False to actually delete these quotes")
        return len(ids_to_delete)
    
    # Actually delete
    print("\n⚠️  DELETING QUOTES...")
    cursor = conn.cursor()
    deleted_count = 0
    total_to_delete = len(ids_to_delete)
    
    for i, quote_id in enumerate(ids_to_delete):
        if i % 100 == 0 or i == total_to_delete - 1:
            progress = (i + 1) / total_to_delete * 100
            sys.stdout.write(f"\r   Deleting: {i + 1}/{total_to_delete} ({progress:.1f}%)")
            sys.stdout.flush()
        
        try:
            cursor.execute("DELETE FROM quotes WHERE id = ?", (quote_id,))
            deleted_count += 1
        except Exception as e:
            print(f"\nError deleting ID {quote_id}: {e}")
    
    print()  # New line after progress
    conn.commit()
    print(f"✓ Successfully deleted {deleted_count} quotes")
    
    return deleted_count

def main():
    db_path = "database.db"
    
    print("=" * 80)
    print("QUOTE SIMILARITY ANALYSIS AND CLEANUP")
    print("=" * 80)
    
    # Connect to database
    conn = connect_db(db_path)
    quotes = get_all_quotes(conn)
    
    print(f"\nTotal quotes in database: {len(quotes)}")
    
    # 1. Find exact duplicates
    print("\n" + "=" * 80)
    print("EXACT DUPLICATES (same text + author)")
    print("=" * 80)
    
    exact_dupes = find_exact_duplicates(quotes)
    if exact_dupes:
        print(f"\nFound {len(exact_dupes)} groups of exact duplicates:\n")
        for i, (key, group) in enumerate(exact_dupes.items(), 1):
            if i <= 5:  # Show only first 5
                print(f"\nGroup {i} ({len(group)} duplicates):")
                for quote in group:
                    print(f"  ID {quote['id']}: {quote['text'][:80]}...")
                    print(f"             Author: {quote['author']}")
        if len(exact_dupes) > 5:
            print(f"\n... and {len(exact_dupes) - 5} more groups")
    else:
        print("\nNo exact duplicates found.")
    
    # 2. Find similar quotes (not exact)
    print("\n" + "=" * 80)
    print("SIMILAR QUOTES (85%+ similarity)")
    print("=" * 80)
    
    similar_groups = find_similar_quotes(quotes, similarity_threshold=0.85)
    if similar_groups:
        print(f"\nFound {len(similar_groups)} groups of similar quotes:\n")
        for i, group in enumerate(similar_groups[:5], 1):  # Show first 5
            main = group['main']
            print(f"\nGroup {i}:")
            print(f"  Main ID {main['id']}: {main['text'][:80]}...")
            print(f"             Author: {main['author']}")
            print(f"\n  Similar to:")
            for sim in group['similar']:
                print(f"    ID {sim['id']} ({sim['similarity']:.1%} match): {sim['text'][:80]}...")
                print(f"               Author: {sim['author']}")
        
        if len(similar_groups) > 5:
            print(f"\n... and {len(similar_groups) - 5} more groups")
    else:
        print("\nNo similar quotes found.")
    
    # 3. Author statistics
    print("\n" + "=" * 80)
    print("QUOTES PER AUTHOR (Top 20)")
    print("=" * 80)
    
    author_stats = analyze_by_author(quotes)
    for i, (author, count) in enumerate(list(author_stats.items())[:20], 1):
        print(f"{i:2d}. {author:50s} - {count:4d} quotes")
    
    print("\n" + "=" * 80)
    print(f"Total unique authors: {len(author_stats)}")
    print("=" * 80)
    
    # 4. Delete duplicates and similar quotes
    print("\n" + "=" * 80)
    print("CLEANUP OPERATION")
    print("=" * 80)
    
    # Ask user for confirmation
    response = input("\nDo you want to delete duplicates and similar quotes? (yes/no): ").strip().lower()
    
    if response == 'yes':
        deleted_count = delete_duplicates_and_similar(conn, dry_run=False)
        print(f"\n✓ Cleanup complete! Deleted {deleted_count} quotes")
        
        # Show new stats
        quotes_after = get_all_quotes(conn)
        print(f"✓ Database now has {len(quotes_after)} quotes (was {len(quotes)})")
    else:
        print("\nRunning in DRY RUN mode (no changes will be made)...")
        delete_duplicates_and_similar(conn, dry_run=True)
    
    conn.close()

if __name__ == "__main__":
    main()