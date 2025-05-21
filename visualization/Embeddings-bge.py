from sentence_transformers import SentenceTransformer
from bertopic.representation import KeyBERTInspired, MaximalMarginalRelevance
from bertopic.dimensionality import BaseDimensionalityReduction
from sklearn.feature_extraction.text import CountVectorizer
from bertopic.vectorizers import ClassTfidfTransformer
import pandas as pd
from bertopic import BERTopic
import numpy as np
import json
import os

# --- Step 1: Prepare your commit messages from a JSON file ---

# --- IMPORTANT: Specify the path to your JSON file ---
json_file_path = '/Users/struewer/Documents/smallmMsg.json' #--- CHANGE THIS

commit_messages = [] # Initialize an empty list to store messages

try:
    print(f"Loading commit messages from {json_file_path}...")
    with open(json_file_path, 'r', encoding='utf-8') as f:
        # Load the entire JSON content
        commit_data = json.load(f)

    # Check if the loaded data is a list and if entries have the 'Message' key
    if isinstance(commit_data, list) and all('Message' in entry for entry in commit_data):
        # Extract the 'Message' from each dictionary in the list
        commit_messages = [entry['Message'].replace("-", " ").lower() for entry in commit_data]
        print(f"Successfully loaded {len(commit_messages)} commit messages.")
    else:
        print(f"Error: JSON data in {json_file_path} is not in the expected format (list of objects with 'Message' key).")
        exit()

except FileNotFoundError:
    print(f"Error: The file {json_file_path} was not found. Please ensure the path is correct.")
    exit()
except json.JSONDecodeError:
    print(f"Error: Could not decode JSON from {json_file_path}. Please check the file format.")
    exit()
except Exception as e:
    print(f"An unexpected error occurred while loading messages: {e}")
    exit()

if not commit_messages:
    print("No commit messages were loaded. Please check your JSON file and the loading logic.")
    exit()



embedding_model_name = "BAAI/bge-large-en-v1.5"
# Pre-calculate embeddings
embedding_model = SentenceTransformer(embedding_model_name)
embeddings = embedding_model.encode(commit_messages, show_progress_bar=True)


np.save('embeddings.npy', embeddings)

print("\nSaving complete.")
