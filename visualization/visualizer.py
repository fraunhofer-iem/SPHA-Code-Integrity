import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import sys
import os

def visualize_commit_data(csv_filepath):
    """
    Reads a CSV file with 'Weekday' and 'Time' columns and generates visualizations.

    Args:
        csv_filepath (str): The path to the input CSV file.
    """
    if not os.path.exists(csv_filepath):
        print(f"Error: CSV file not found at {csv_filepath}", file=sys.stderr)
        sys.exit(1)

    try:
        # Read the CSV file into a pandas DataFrame
        df = pd.read_csv(csv_filepath)
    except Exception as e:
        print(f"Error reading CSV file {csv_filepath}: {e}", file=sys.stderr)
        sys.exit(1)

    # --- Data Preparation ---

    # Ensure required columns exist
    if 'Weekday' not in df.columns or 'Time' not in df.columns:
        print("Error: CSV must contain 'Weekday' and 'Time' columns.", file=sys.stderr)
        sys.exit(1)

    # Handle potential empty DataFrame
    if df.empty:
        print("Warning: The CSV file is empty. No data to visualize.", file=sys.stderr)
        return

    # Define the correct order for weekdays for consistent plotting
    weekday_order = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday']

    # Convert 'Weekday' to a categorical type with the specified order
    df['Weekday'] = pd.Categorical(df['Weekday'], categories=weekday_order, ordered=True)

    # Extract the hour from the 'Time' column
    # Assumes 'Time' is in HH:MM:SS format
    try:
        df['Hour'] = pd.to_datetime(df['Time'], format='%H:%M:%S').dt.hour
    except ValueError as e:
         print(f"Error parsing 'Time' column. Ensure it's in HH:MM:SS format: {e}", file=sys.stderr)
         sys.exit(1)


    # --- Visualization ---

    # Set a style for the plots
    sns.set_theme(style="whitegrid")

    # Chart 1: Distribution of Days
    plt.figure(figsize=(10, 6))
    sns.countplot(y='Weekday', data=df, order=weekday_order, palette='viridis')
    plt.title('Distribution of Commits by Day of the Week')
    plt.xlabel('Number of Commits')
    plt.ylabel('Weekday')
    plt.tight_layout() # Adjust layout to prevent labels overlapping

    # Chart 3: Distribution of Time, Clustered by Hour (Overall)
    plt.figure(figsize=(12, 6))
    # Get the unique hours and sort them for correct plotting order
    hour_order = sorted(df['Hour'].unique())
    sns.countplot(x='Hour', data=df, order=hour_order, palette='viridis')
    plt.title('Overall Distribution of Commits by Hour of Day')
    plt.xlabel('Hour of Day (0-23)')
    plt.ylabel('Number of Commits')
    plt.xticks(rotation=0) # Ensure hour labels are horizontal
    plt.tight_layout()

    # Chart 2: Distribution of Time, Connected to Days (Hourly by Weekday)
    # Use seaborn's catplot to create separate plots for each weekday
    # This is often cleaner than trying to cram all days onto one chart as grouped bars
    g = sns.catplot(
        x='Hour',
        col='Weekday',
        data=df,
        kind='count',
        col_wrap=4, # Display 4 plots per row
        height=4,   # Height of each facet
        aspect=.7,  # Aspect ratio of each facet
        palette='viridis',
        col_order=weekday_order # Ensure columns (weekdays) are in order
    )
    g.set_axis_labels("Hour of Day", "Number of Commits")
    g.set_titles("{col_name}") # Set title for each subplot (e.g., "Weekday: Monday")
    g.fig.suptitle("Hourly Commit Distribution by Weekday", y=1.03) # Add a main title
    plt.tight_layout(rect=[0, 0.03, 1, 0.98]) # Adjust layout for suptitle


    # Display all the plots
    plt.show()

if __name__ == "__main__":
    # Example usage: replace 'all_commit_times.csv' with your generated file path
    # You could also modify this to accept the file path as a command-line argument
    csv_file = 'all_commit_times.csv'

    # Check if a command-line argument is provided for the file path
    if len(sys.argv) > 1:
        csv_file = sys.argv[1]

    visualize_commit_data(csv_file)
