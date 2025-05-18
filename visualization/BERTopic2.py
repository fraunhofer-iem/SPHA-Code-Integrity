import pandas as pd
from bertopic import BERTopic
import numpy as np
import json
import os # Import os for creating directories

# --- Step 1: Prepare your commit messages from a JSON file ---

# --- IMPORTANT: Specify the path to your JSON file ---
json_file_path = '/Users/struewer/Documents/allMsg.json' #--- CHANGE THIS

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
topic_model = BERTopic(verbose=True)

# --- Step 3: Fit the model to your commit messages ---
print("\nFitting BERTopic model...")
topics, probs = topic_model.fit_transform(commit_messages)
print("Model fitting complete.")

# --- Step 4: Get and display the topics ---
print("\nDiscovered Topics:")
most_frequent_topics = topic_model.get_topic_info()
print(most_frequent_topics)

print(f"\nTotal number of topics found (including outliers): {len(topic_model.get_topics())}")
print(f"Number of outlier messages (-1 topic): {list(topics).count(-1)}")


# --- Step 5: Save the results ---

# Define a directory to save results
output_dir = "bertopic_results"
os.makedirs(output_dir, exist_ok=True) # Create the directory if it doesn't exist

print(f"\nSaving results to '{output_dir}/'...")

# 5a. Save the BERTopic model
# This saves the model and its components (embeddings, clustering, etc.)
model_save_path = os.path.join(output_dir, "bertopic_model")
topic_model.save(model_save_path, serialization="safetensors") # Using safetensors is recommended

print(f"BERTopic model saved to {model_save_path}")

# 5b. Save the topic assignments for each message
# We can create a simple DataFrame mapping original messages to topics
# Note: For 2 million messages, saving the original messages might create a very large file.
# You might prefer to save just the topic assignments indexed by their original order,
# or combine topic assignments with GitOID from your JSON if you loaded that too.
# For simplicity, let's save message + topic here.
results_df = pd.DataFrame({'Message': commit_messages, 'Topic': topics})
assignments_save_path = os.path.join(output_dir, "commit_topic_assignments.csv")
results_df.to_csv(assignments_save_path, index=False)

print(f"Topic assignments saved to {assignments_save_path}")

# 5c. Save the topic information summary
topic_info_save_path = os.path.join(output_dir, "topic_information.csv")
most_frequent_topics.to_csv(topic_info_save_path, index=False)

print(f"Topic information summary saved to {topic_info_save_path}")

print("\nSaving complete.")

# --- Optional: How to load the model later ---
# To load the saved model in a new script or session:
# loaded_model = BERTopic.load(model_save_path)
# print("\nLoaded BERTopic model successfully.")
# print(loaded_model.get_topic_info()) # You can now use the loaded model
