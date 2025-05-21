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

# Attempt to import cuML components
try:
    from cuml.manifold import UMAP as cumlUMAP
    from cuml.cluster import HDBSCAN as cumlHDBSCAN
    use_cuml = True
    print("cuML found. Will attempt to use GPU for UMAP and HDBSCAN.")
except ImportError:
    use_cuml = False
    print("cuML not found. UMAP and HDBSCAN will run on CPU.")
    # You might want to fall back to CPU versions or raise an error
    from umap import UMAP
    from hdbscan import HDBSCAN


# Define UMAP and HDBSCAN models
if use_cuml:
    umap_model = cumlUMAP(n_neighbors=15, n_components=5, min_dist=0.0, random_state=42)
    # Ensure hdbscan_model parameters are compatible with cuML's HDBSCAN
    # cuML's HDBSCAN might have slightly different parameter names or defaults
    # For example, `prediction_data=True` might be needed if you want to predict topics for new data later with cuML
    hdbscan_model = cumlHDBSCAN(min_cluster_size=20, min_samples=1,
                                metric='euclidean', gen_min_span_tree=True,
                                prediction_data=True) # prediction_data=True is often needed for later use
else:
    umap_model = UMAP(n_neighbors=15, n_components=5, min_dist=0.0, metric='cosine', random_state=42)
    hdbscan_model = HDBSCAN(min_cluster_size=20, metric='euclidean',
                            cluster_selection_method='eom', prediction_data=True)


embedding_model_name = "BAAI/bge-large-en-v1.5"
# Pre-calculate embeddings
embedding_model = SentenceTransformer(embedding_model_name)
embeddings = embedding_model.encode(commit_messages, show_progress_bar=True)
print(embeddings)
# MMR
mmr = MaximalMarginalRelevance(diversity=0.3)
keybert = KeyBERTInspired()

# All representation models
representation_model = {
    "KeyBERT": keybert,
    "MMR": mmr,
}

vectorizer_model = CountVectorizer(stop_words="english", ngram_range=(1, 3),)

ctfidf_model = ClassTfidfTransformer(reduce_frequent_words=True)

topic_model = BERTopic(
    vectorizer_model=vectorizer_model,
    umap_model= umap_model,
    hdbscan_model=hdbscan_model,
    verbose=True,
    embedding_model=embedding_model, # Uncomment if you want to specify one
    representation_model=representation_model,
    ctfidf_model=ctfidf_model,
    nr_topics="auto",
)



# --- Step 3: Fit the model to your commit messages ---
# ... (rest of your code remains the same) ...
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
output_dir = "bertopic_results"
os.makedirs(output_dir, exist_ok=True)
print(f"\nSaving results to '{output_dir}/'...")
model_save_path = os.path.join(output_dir, "bertopic_model")
topic_model.save(model_save_path, serialization="safetensors")
print(f"BERTopic model saved to {model_save_path}")

results_df = pd.DataFrame({'Message': commit_messages, 'Topic': topics})
assignments_save_path = os.path.join(output_dir, "commit_topic_assignments.csv")
results_df.to_csv(assignments_save_path, index=False)
print(f"Topic assignments saved to {assignments_save_path}")

topic_info_save_path = os.path.join(output_dir, "topic_information.csv")
most_frequent_topics.to_csv(topic_info_save_path, index=False)
print(f"Topic information summary saved to {topic_info_save_path}")
output_dir = "bertopic_results"

topic_model.save(output_dir, serialization="safetensors", save_ctfidf=True, save_embedding_model=embedding_model_name)
print("\nSaving complete.")
