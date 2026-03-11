# Contributor guide

This document lays out the best practices and other rules for contributing to this repository.

## Git best practices

This repository follows [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) to ensure clean Git history. The following types are supported:

- **feat:** Add a new feature.
- **fix:** Fix a bug.
- **refactor:** Code changes that are neither a fix nor a feature.
- **chore:** Update repository settings such as lint configs.
- **build:** Update the build configuration.
- **ci:** Changes to CI/CD pipelines.
- **docs:** Changes to the documentation.
- **perf:** Performance improvements.
- **test:** Add or update tests.

Branches should also be prefixed with the type of change they are to introduce, for example a new feature branch would be `feat/my-new-feature`.

### Committing

This repository is configured with a commit message linter that will enforce the above style. Make sure that you have run `just init` so the pre-commit hooks are installed. This will ensure that all your commits conform to these standards.

If you wish to create commits that do not conform to conventional commits, you can use `git commit -Nm "work on branch"` to disable the checks. Note that `-N` is shorthand for `--no-verify`. Once your changes are ready to upstream you can use `git rebase -i HEAD^N`, where `N` is the number of commits to go back, to edit commit messages and merge commits together. You can also use a commit SHA as the rebase target. For this workflow, it is highly recommend that you familiarize yourself with your IDE's git tooling or a dedicated tool such as Tower. Dealing with commit logs, merges, and interactive rebases are considerably easier in a graphical tool.

### Pushing

Before pushing changes, this repository is configured to lint your changes. This will prevent potentially bad code from being pushed to the remote. Like committing, you can disable these checks with the `--no-verify` flag. **This should never be done for `main`, only development branches.**

### Pull requests

Most work should be done on development branches that are opened for review before merging into `main`. Observe the following best practices when creating these requests:

- Use draft pull requests for WIP pull requests.
- Open draft pull requests early so team members can see active work.
- Do not request approval until all CI checks are passing.
- Do not force push once you have marked the pull request ready for review.
- Use the review feature to group comments.
- Put technical details in commit messages and business context in the pull request.
- Enable auto-merge once your pull request is ready for review.
- Use merge commits when the pull request has discussion or other context.
- Use rebase commits when the pull request is empty or generated.

## Editor configuration

Most editors will have Go and React/TypeScript plugins. For example, when using Visual Studio Code, the following plugins should be present:

- [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go)
- [ESLint](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint)
- [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode)
- [EditorConfig](https://marketplace.visualstudio.com/items?itemName=EditorConfig.EditorConfig)
