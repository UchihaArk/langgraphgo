import sys
from markitdown import MarkItDown

if len(sys.argv) != 2:
    print("Usage: python convert_webpage.py <URL>")
    sys.exit(1)

url = sys.argv[1]

try:
    md = MarkItDown()
    result = md.convert(url)
    print(result.text_content)
except Exception as e:
    print(f"An error occurred: {e}")
    sys.exit(1)
