Language selection based on [GitHub Blog](https://github.blog/news-insights/octoverse/octoverse-2024) of most used languages.

Search query for GitHub API to get the 100 most starred projects for each language:
```
query {
search(query: "language:$language sort:stars-desc", type:REPOSITORY, first:100) {
  nodes { ... on Repository {
    nameWithOwner
    stargazerCount
    url
  }}
}
}
```

Data querried on 17.04.2025.
