# Conventional Commits 1.0.0 Summary

Conventional Commits define a standard format for commit messages:

type(scope)!: summary

Optional body and footer sections follow a blank line.

Key rules:
- type is required
- scope is optional
- ! indicates a breaking change
- use BREAKING CHANGE: in the footer to describe breaking changes

Examples:
- feat(auth): add token refresh
- fix!: drop legacy login
  
  BREAKING CHANGE: legacy login removed
