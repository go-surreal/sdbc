# Contributing

We would love for you to contribute to **sdbc** and help make it better!
We want contributions to be fun, enjoyable, and educational for anyone and everyone.
All contributions are welcome, including features, bugfixes, and documentation changes.

## How to start

tbd.

## Code of conduct

Help us keep sdbc open and inclusive. Please read and follow our [Code of Conduct](/CODE_OF_CONDUCT.md).

## Coding standards

- go fmt
- golangci-lint
- ci workflows

## Performance

tbd.

## Security and Privacy

tbd.

## External dependencies

Please avoid introducing new dependencies to sdbc without consulting the team.
New dependencies can be very helpful but also introduce new security and privacy issues as well as complexity.

## Introducing new features

While it is always great to have new contributions, please acknowledge that this project tries to stay close to what an official SurrealDB client/driver would support.
Anything that goes beyond the scope of that might be better suited for a separate project.
For us to find the right balance, please open a question on [GitHub discussions](https://github.com/go-surreal/sdbc/discussions) with any ideas before introducing a new pull request.
This will allow the community to have sufficient discussion about the new feature value and how it fits in the roadmap and overall vision.

We might introduce an RFC process in the future to formalize this process.

## Submitting a pull request

Branch naming convention is as following

`TYPE-ISSUE_ID-DESCRIPTION`

For example:
```
bugfix-548-ensure-queries-execute-sequentially
```

Where `TYPE` can be one of the following:

- **refactor** - code change that neither fixes a bug nor adds a feature
- **feature** - code changes that add a new feature
- **bugfix** - code changes that fix a bug
- **docs** - documentation only changes
- **ci** - changes related to CI system

For the initial start, fork the project and use git clone command to download the repository to your computer. A standard procedure for working on an issue would be to:

1. Clone the `sdbc` repository and download to your computer.
    ```bash
    git clone https://github.com/go-surreal/sdbc
    ```

   (Optional): Install [pre-commit](https://pre-commit.com/#install) to run the checks before each commit and run:

    ```bash
    pre-commit install
    ```

2. Pull all changes from the upstream `main` branch, before creating a new branch - to ensure that your `main` branch is up-to-date with the latest changes:
    ```bash
    git pull
    ```

3. Create new branch from `main` like: `bugfix-548-ensure-queries-execute-sequentially`:
    ```bash
    git checkout -b "[the name of your branch]"
    ```

4. Make changes to the code, and ensure all code changes are formatted correctly:
    ```bash
    go fmt
    ```

5. Commit your changes when finished:
    ```bash
    git add -A
    git commit -m "[your commit message]"
    ```

6. Push changes to GitHub:
    ```bash
    git push origin "[the name of your branch]"
    ```

7. Submit your changes for review, by going to your repository on GitHub and clicking the `Compare & pull request` button.

8. Ensure that you have entered a commit message which details about the changes, and what the pull request is for.

9. Now submit the pull request by clicking the `Create pull request` button.

10. Wait for code review and approval.

11. After approval, merge your pull request.

## Other Ways to Help

Pull requests are great, but there are many other areas where you can help.

### Feedback, bugs, and ideas

tbd.

### Documentation improvements

tbd.

### Joining the SurrealDB community

Join the growing community of SurrealDB!

- View the official [Blog](https://surrealdb.com/blog)
- Follow them on [Twitter](https://twitter.com/surrealdb)
- Connect with them on [LinkedIn](https://www.linkedin.com/company/surrealdb/)
- Join their [Dev community](https://dev.to/surrealdb)
- Chat live with them on [Discord](https://discord.gg/surrealdb)
- Get involved on [Reddit](http://reddit.com/r/surrealdb/)
- Read their blog posts on [Medium](https://medium.com/surrealdb)
- Questions tagged #surrealdb on [StackOverflow](https://stackoverflow.com/questions/tagged/surrealdb)
