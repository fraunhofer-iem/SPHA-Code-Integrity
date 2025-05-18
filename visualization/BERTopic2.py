import pandas as pd
from bertopic import BERTopic
import numpy as np
import json # Import the json library

# --- Step 1: Prepare your commit messages from a JSON file ---

# --- IMPORTANT: Specify the path to your JSON file ---
json_file_path = '/Users/struewer/Documents/allMsg.json' # <--- CHANGE THIS to the actual path

commit_messages = [] # Initialize an empty list to store messages

try:
    print(f"Loading commit messages from {json_file_path}...")
    with open(json_file_path, 'r', encoding='utf-8') as f:
        # Load the entire JSON content
        commit_data = json.load(f)

    # Check if the loaded data is a list and if entries have the 'Message' key
    if isinstance(commit_data, list) and all('Message' in entry for entry in commit_data):
        # Extract the 'Message' from each dictionary in the list
        commit_messages = [entry['Message'] for entry in commit_data]
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

# --- Step 2: Initialize BERTopic model ---
# For a large dataset like 2 million, consider adjusting parameters.
# For example, min_topic_size might need to be larger than the default (10).
# You might also explore different embedding models.
# The 'verbose=True' option provides progress updates.
topic_model = BERTopic(verbose=True)

# --- Step 3: Fit the model to your commit messages ---
# This is the main step where embeddings are generated, dimensionality reduction
# is applied, and clustering is performed.
print("\nFitting BERTopic model...")
topics, probs = topic_model.fit_transform(commit_messages)
print("Model fitting complete.")

# --- Step 4: Get and display the topics ---
# -1 usually represents outliers (messages that don't clearly belong to any topic)
print("\nDiscovered Topics:")
# Get the most frequent topics (excluding outliers)
most_frequent_topics = topic_model.get_topic_info()
print(most_frequent_topics)

# You can also get the representative documents for a specific topic
# For example, for topic 0 (if it exists and is not the outlier topic -1)
# if 0 in most_frequent_topics['Topic'].values:
#     print(f"\nRepresentative documents for Topic 0:")
#     print(topic_model.get_document_info(commit_messages)[topic_model.get_document_info(commit_messages)['Topic'] == 0]['Document'].tolist())

# You can get the words for a specific topic
# For example, for topic 0 (if it exists and is not the outlier topic -1)
# if 0 in most_frequent_topics['Topic'].values:
#      print(f"\nWords for Topic 0:")
#      print(topic_model.get_topic(0))

print(f"\nTotal number of topics found (including outliers): {len(topic_model.get_topics())}")
print(f"Number of outlier messages (-1 topic): {list(topics).count(-1)}")

# --- Optional: Reduce topics if needed ---
# If you end up with too many topics, you can reduce them.
# new_topics, new_probs = topic_model.reduce_topics(commit_messages, nr_topics=50)
# print(f"\nReduced to {len(topic_model.get_topics())} topics.")
# print(topic_model.get_topic_info())
