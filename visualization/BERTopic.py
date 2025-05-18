import pandas as pd
from bertopic import BERTopic
import numpy as np # Often useful, though not strictly necessary for basic BERTopic fit/transform

# --- Step 1: Prepare your commit messages ---
# Replace this list with your actual commit messages.
# For 2 million messages, you'll likely load these from a file or a database.
# Example placeholder list:
# commit_messages = [
#     "feat: Add new user registration feature",
#     "fix: Correct typo in login page",
#     "refactor: Improve performance of data fetching",
#     "docs: Update README with installation instructions",
#     "test: Add unit tests for API endpoints",
#     "feat: Implement dark mode toggle",
#     "fix: Resolve issue with infinite loop in parser",
#     "refactor: Clean up unused variables in main module",
#     "docs: Add contributing guidelines",
#     "test: Write integration tests for database operations",
#     "chore: Update dependencies",
#     "feat: Add search functionality",
#     "fix: Fix styling issue on mobile view",
# ]

# --- IMPORTANT: How to load your 2 million messages ---
# You will need to replace the small placeholder list above with a method
# to load your large dataset of commit messages.
# For example, if your messages are in a file named 'commit_messages.txt',
# with one message per line, you could load them like this:
# try:
#     with open('commit_messages.txt', 'r', encoding='utf-8') as f:
#         commit_messages = [line.strip() for line in f if line.strip()]
#     print(f"Loaded {len(commit_messages)} commit messages.")
# except FileNotFoundError:
#     print("Error: commit_messages.txt not found. Please create this file or provide your messages.")
#     exit()
# except Exception as e:
#     print(f"An error occurred while loading messages: {e}")
#     exit()

# --- Placeholder for demonstration ---
# Using a small list for demonstration purposes.
# REMEMBER TO REPLACE THIS WITH YOUR ACTUAL DATA LOADING!
commit_messages = [
    "feat: Add new user registration feature",
    "fix: Correct typo in login page",
    "refactor: Improve performance of data fetching",
    "docs: Update README with installation instructions",
    "test: Add unit tests for API endpoints",
    "feat: Implement dark mode toggle",
    "fix: Resolve issue with infinite loop in parser",
    "refactor: Clean up unused variables in main module",
    "docs: Add contributing guidelines",
    "test: Write integration tests for database operations",
    "chore: Update dependencies",
    "feat: Add search functionality",
    "fix: Fix styling issue on mobile view",
    "build: Add Dockerfile for deployment",
    "ci: Configure GitHub Actions for testing",
    "perf: Optimize database queries",
    "feat: Implement password reset functionality",
    "fix: Address security vulnerability",
    "refactor: Simplify error handling logic",
    "docs: Generate API documentation",
    "test: Mock external API calls",
    "chore: Clean up temporary files",
    "feat: Add user profile page",
    "fix: Correct off-by-one error",
    "refactor: Extract common functionality into a helper module",
    "docs: Add examples to documentation",
    "test: Test edge cases",
    "build: Update build script",
    "ci: Fix failing build on main branch",
    "perf: Cache frequently accessed data",
]


if not commit_messages:
    print("No commit messages to process. Please ensure you have loaded your messages.")
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
