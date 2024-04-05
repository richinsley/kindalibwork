import sys
print(sys.path)
    
from quote import quote

def print_random_quote(category='inspirational'):
    try:
        res = quote('family',limit=1)
        quotes = quote(category,limit=1)
        if quotes:
            print(f"\"{quotes[0]['quote']}\" - {quotes[0]['author']}")
        else:
            print("No quotes found for the given category.")
    except Exception as e:
        print(f"An error occurred: {e}")

if __name__ == "__main__":
    print("here's your stupid quote:")
    if len(sys.argv) > 1:
        category = sys.argv[1]
    else:
        category = 'inspirational'
    print_random_quote(category)