# Contributing to MetaHub Registry

Thanks for taking the time to contribute. This document covers the mechanics;
the [README](README.md) covers what the project is and how to run it.

## Ground rules

- Be respectful. The project follows the [Code of Conduct](CODE_OF_CONDUCT.md).
- Security issues are **not** public issues. See [SECURITY.md](SECURITY.md).
- By contributing, you agree that your contribution is licensed under the
  Apache License 2.0, the same licence as the project.

## Developer Certificate of Origin (DCO)

Every commit must carry a `Signed-off-by` trailer certifying the
[Developer Certificate of Origin](https://developercertificate.org/):

```bash
git commit -s -m "fix(api): handle empty profile suffix"
```

The sign-off must use your real name and an email you can receive mail at. CI
rejects pull requests whose commits are not signed off.

There is no Contributor Licence Agreement. Contributions stay under
Apache-2.0 and cannot be relicensed into a proprietary edition; this is
deliberate and documented in [GOVERNANCE.md](GOVERNANCE.md).

## Before you start

- For a bug, open an issue with the bug-report template.
- For a feature, open an issue first so we can agree on the shape before you
  write code. Small fixes can go straight to a pull request.
- Check the [roadmap](ROADMAP.md) for what is already planned.

## Development

1. Fork the repository and create a topic branch.
2. Make a focused change. Avoid drive-by reformatting; it makes review harder.
3. Add or update tests. A bug fix should have a test that fails without it.
4. Update the documentation if behaviour or configuration changes.
5. Run the repository's test command (`./test.sh` for Go repositories,
   `npm test` for the frontends) and report the result in the pull request.
6. Open the pull request using the template, with the problem and the approach
   described.

### Commit messages

Use Conventional Commits, matching the existing history:

```
feat(inventory): resolve profile from hostname
fix(auth): reject empty bearer token
docs(readme): explain the offline quickstart
chore(deps): bump neo4j driver
```

One logical change per commit. Keep the subject under ~72 characters.

### Code style

- Go: `gofmt`, `go vet`. No new linters without discussion.
- TypeScript/React: ESLint and the repository's formatter.
- YAML/docs: 2-space indentation, LF line endings (see `.editorconfig`).

## Review

Reviews are best effort by the maintainers; there is no service-level
agreement. Small pull requests with tests and a clear description are reviewed
fastest. A maintainer may ask you to split a change.

## Reporting a security issue

Do not open a public issue. Follow [SECURITY.md](SECURITY.md).

## Code of conduct

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Report
concerns to `metahub@qnib.org`.
