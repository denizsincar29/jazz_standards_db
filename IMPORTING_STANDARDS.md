# Guide: Importing 2000+ Jazz Standards

This guide provides instructions and resources for importing a comprehensive collection of jazz standards into your database.

## Overview

There are several approaches to building a database of 2000+ jazz standards. This guide covers legal and ethical methods to collect and import this data.

## Legal Considerations

⚠️ **IMPORTANT**: When importing jazz standards, only include:
- **Title** (not copyrightable)
- **Composer name** (factual information)
- **Style/Genre** (factual information)
- **Basic metadata** (year, key signature, etc.)

❌ **DO NOT INCLUDE**:
- Copyrighted chord progressions/charts
- Lyrics
- Musical notation
- Audio recordings

## Data Sources

### 1. Wikipedia - List of Jazz Standards ⭐ (Recommended)

Wikipedia maintains a comprehensive list of jazz standards that is publicly accessible.

**Source**: https://en.wikipedia.org/wiki/List_of_jazz_standards

**Data Available**:
- Song titles
- Composers
- Years written
- Original sources

**How to Extract**:
```python
# Example Python script to extract from Wikipedia
import requests
from bs4 import BeautifulSoup
import json

url = "https://en.wikipedia.org/wiki/List_of_jazz_standards"
response = requests.get(url)
soup = BeautifulSoup(response.content, 'html.parser')

standards = []
for row in soup.find_all('tr')[1:]:  # Skip header
    cols = row.find_all('td')
    if len(cols) >= 2:
        title = cols[0].text.strip()
        composer = cols[1].text.strip()
        standards.append({
            "title": title,
            "composer": composer,
            "style": "swing"  # Default, adjust as needed
        })

with open('standards.json', 'w') as f:
    json.dump(standards, f, indent=2)
```

### 2. The Real Book Index

The Real Book is a widely-used collection of jazz standards. While the charts are copyrighted, the **list of titles and composers** is factual information.

**Sources**:
- Real Book (5th & 6th editions)
- New Real Book series
- Standards Real Book

**Manual Compilation**: Create a spreadsheet with titles and composers from the table of contents.

### 3. JazzStandards.com

Website with a searchable database of standards (check robots.txt and terms of service before scraping).

### 4. AllMusic Jazz Glossary

Contains lists and information about jazz standards organized by era and style.

### 5. Public Domain Sheet Music Sites

- **IMSLP (Petrucci Music Library)**: Public domain scores
- **Library of Congress**: Historical recordings metadata

### 6. Community Contributions

Build the database collaboratively with your users:
- Allow users to submit standards (with moderation)
- Verify submissions against authoritative sources
- Gradually build comprehensive collection

## Data Format

Use JSON format for easy importing:

```json
[
  {
    "title": "Autumn Leaves",
    "composer": "Joseph Kosma",
    "style": "swing",
    "additional_note": "Originally 'Les Feuilles Mortes'"
  },
  {
    "title": "All the Things You Are",
    "composer": "Jerome Kern",
    "style": "swing",
    "additional_note": "From 'Very Warm for May'"
  }
]
```

## Style Classification

Categorize standards into these styles:
- `swing` - Pre-1950s standards
- `bebop` - 1940s-1950s
- `bossa_nova` - Brazilian influenced
- `latin` - Latin jazz
- `modal` - Modal jazz (1960s)
- `fusion` - Jazz fusion (1970s+)
- `dixieland` - Early jazz
- `ragtime` - Pre-jazz era
- `big_band` - Big band era
- `waltz` - 3/4 time signatures
- `free` - Free jazz

## Import Process

### Step 1: Prepare Your Data

Create a JSON file `standards.json`:

```json
[
  {"title": "...", "composer": "...", "style": "swing"},
  ...
]
```

### Step 2: Get Admin Token

First, create an admin account:

```bash
curl -X POST http://localhost:8000/api/admin \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","name":"Admin","password":"secure-password"}'
```

Save the returned token.

### Step 3: Run Import Script

```bash
# Set your admin token
export ADMIN_TOKEN="your-admin-token-here"

# Run the import
go run scripts/import_standards.go standards.json
```

### Step 4: Monitor Import

The script will output progress:
```
Found 2000 standards to import
[1/2000] Importing: Autumn Leaves - Joseph Kosma
  ✓ Success
[2/2000] Importing: All the Things You Are - Jerome Kern
  ✓ Success
...
Import complete:
  Success: 1950
  Errors:  50
  Total:   2000
```

## Sample Data Sources

### Pre-compiled Lists

1. **Wikipedia Jazz Standards** (~1000 standards)
   - Most comprehensive public source
   - Well-maintained
   - Includes metadata

2. **Great American Songbook**
   - Classic standards (1920s-1950s)
   - ~500 core standards
   - Freely available lists

3. **Public Domain Songs Database**
   - Pre-1925 songs (public domain)
   - ~200-300 jazz-era songs

## Building to 2000+ Standards

### Recommended Approach

1. **Start with Wikipedia** (~1000 standards)
   ```bash
   # Extract from Wikipedia
   python scripts/extract_wikipedia.py
   ```

2. **Add Real Book Titles** (~400 additional)
   - Manually compile from table of contents
   - Focus on unique titles not in Wikipedia

3. **Include Bebop Compositions** (~300)
   - Charlie Parker tunes
   - Dizzy Gillespie compositions
   - Thelonious Monk standards

4. **Add Bossa Nova/Latin** (~200)
   - Antonio Carlos Jobim
   - João Gilberto
   - Latin jazz standards

5. **Include Modern Standards** (~100)
   - Pat Metheny compositions
   - Herbie Hancock tunes
   - Modern jazz composers

**Total**: ~2000 standards

## Quality Control

### Deduplication

```bash
# Remove duplicates from your JSON
jq 'unique_by(.title)' standards.json > standards_unique.json
```

### Validation

Before importing, validate your data:

```bash
# Check for required fields
jq '.[] | select(.title == null or .composer == null or .style == null)' standards.json
```

### Style Verification

Ensure all styles are valid:

```bash
# Valid styles
grep -o '"style": "[^"]*"' standards.json | sort | uniq
```

## Maintenance

### Regular Updates

1. **Weekly**: Check Wikipedia for updates
2. **Monthly**: Review user submissions
3. **Quarterly**: Add new publications

### Community Moderation

Allow trusted users to:
- Flag incorrect information
- Suggest additions
- Vote on disputed entries

## Example: Complete Import Workflow

```bash
# 1. Prepare environment
export ADMIN_TOKEN="your-token"
export API_URL="http://localhost:8000"

# 2. Download Wikipedia data
python scripts/extract_wikipedia.py > wikipedia_standards.json

# 3. Manually add Real Book titles
cat realbook.json wikipedia_standards.json | jq -s 'add' > combined.json

# 4. Deduplicate
jq 'unique_by(.title)' combined.json > standards_final.json

# 5. Validate
jq '. | length' standards_final.json  # Should show ~2000

# 6. Import
go run scripts/import_standards.go standards_final.json

# 7. Verify in database
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8000/api/jazz_standards | jq '. | length'
```

## Advanced: Web Scraping

If you need to scrape data, **always**:

1. Check `robots.txt`
2. Respect rate limits
3. Add delays between requests
4. Identify your bot in User-Agent
5. Only scrape public data

Example responsible scraper:

```python
import time
import requests

def scrape_responsibly(url):
    headers = {
        'User-Agent': 'JazzStandardsBot/1.0 (Educational Purpose)'
    }
    time.sleep(2)  # 2 second delay
    return requests.get(url, headers=headers)
```

## Resources

### Books (for manual compilation)
- The Real Book (Hal Leonard)
- The New Real Book (Sher Music)
- The Standards Real Book
- Jazz Standards for Singers

### Websites
- JazzStandards.com
- AllAboutJazz.com
- Wikipedia Jazz portals
- Internet Archive (public domain recordings)

### Academic Sources
- Smithsonian Institution jazz archives
- Library of Congress jazz collections
- University jazz libraries

## Legal Notice

This guide is for educational purposes. When building a database:

1. **Respect Copyright**: Only use factual information (titles, composers)
2. **Attribution**: Credit your sources
3. **Terms of Service**: Comply with website terms
4. **Fair Use**: Ensure your use qualifies as fair use
5. **Consult Legal**: When in doubt, consult a lawyer

## Troubleshooting

### Import Errors

**Duplicate titles**:
```
✗ Failed (status 409): Standard already exists
```
Solution: Check for exact title match, or skip duplicates

**Invalid style**:
```
✗ Failed (status 400): Invalid style
```
Solution: Use only valid styles listed in this guide

**Rate limiting**:
```
✗ Failed (status 429): Too many requests
```
Solution: Increase delay in import script

## Contributing

Help expand this guide:
1. Share your data sources
2. Contribute import scripts
3. Report inaccuracies
4. Suggest improvements

Open an issue or submit a pull request!

## Questions?

- **Need help?** Open an issue on GitHub
- **Found a better source?** Share it with the community
- **Legal concerns?** Consult appropriate legal counsel

---

**Remember**: Quality over quantity. 500 accurate, well-categorized standards are better than 2000 dubious entries!
