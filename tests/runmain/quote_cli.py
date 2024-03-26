import sys
# site_packages_path = "/Users/richardinsley/Projects/comfycli/kindalib/tests/micromamba/envs/myenv3.10/lib/python3.10/site-packages"  # Adjust if needed
# if site_packages_path not in sys.path:
#     sys.path.append(site_packages_path)
    
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