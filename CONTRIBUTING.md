# Contributing

Thanks for taking an interest in contributing to LeifDB! This guide is meant to make it easy to get involved.

## What to work on

Contributions of all kinds are welcome. That might mean adding a bug report or feature request, getting involved in a discussion, or contributing code. If you don't know where to start, take a look at the [Issues](https://github.com/btmorr/leifdb/issues), which are also organized into [Milestones](https://github.com/btmorr/leifdb/milestones) representing discrete features that are larger than a single ticket. You're also welcome to submit ideas other than what's already there. If you have a question, go ahead and file an issue.

## Opening an issue

Feature requests and bug reports can both be submitted by opening [an Issue in this repo](https://github.com/btmorr/leifdb/issues).

## Opening pull requests

If you have an improvement that you'd like to share, pull requests are appreciated!

You may already have a developer environment set up. If not, check out the [section below](#developer-environment) for recommendations.

Is the PR related to an open issue?

- If so, [link the PR to the issue](https://help.github.com/en/github/managing-your-work-on-github/linking-a-pull-request-to-an-issue#linking-a-pull-request-to-an-issue-using-a-keyword), such as "Closes #123"--this will make it so that when the PR is merged, the issue will be closed automatically
- If not, please include an explanation of the goal of the PR in the description (ideally including the info you would put in an issue describing the feature or fix that is the subject of the PR)

Make sure that your PR branch is up to date with the base branch (or, at least, that there are no conflicting changes on the lines you're modifying). Also remember to format, document, and test code to make sure the PR is ready to go. You're absolutely welcome to open a PR that is not completely ready yet in order to ask questions or ask for assitance--consider opening a [draft PR](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/about-pull-requests#draft-pull-requests) and then notifying one of the [maintainers](#maintainers) with an @-mention in the PR comments.

Once you have submitted a PR and it is marked "Ready for review", one of the [maintainers](#maintainers) will take a look as soon as possible (thank you for your patience, and feel free to mention us again in the PR comments if we haven't responded in a few days). We'll either provide feedback with questions or directions on what needs to be done to be ready to merge (or an explanation if the change is not adopted), or we'll approve and merge it.

Finally, remember to add yourself to the [contributors section below](#contributors), and thanks again for getting involved!

[Please note that by contributing to this project, your contributions will be licensed under [the project's license](./LICENSE)]

## Developer environment

### Build and run environment

Go v1.14 is the only strict requirement for this project. You can find instructions on getting and installing Go [here](https://golang.org/dl/).

If you want to change something in a probobuf definition, you'll also need protoc and the protoc-gen-go plugin.

To install protoc, download the package for your OS [here](https://github.com/protocolbuffers/protobuf/releases/), unzip it, and move the protoc binary from the resulting "bin" directory into \$GOPATH/bin. Then run this command to install the plugin to generate Go code from a .proto file:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

If you have trouble finding the binary, on a Unix or Linux system you can try:

```
find $(go env GOPATH) -name protoc-gen-go
```

Additionally this repo includes a Makefile with tasks for common workflows. Your OS may already have [make](https://www.gnu.org/software/make/) installed (you can check on a Linux/Unix system using `which make` or on Windows using `Get-Command make` in PowerShell). If you don't have it installed, you can get it via a package manager (ex: `yum` for Centos, `apt` for Debian/Ubuntu, [`brew`](https://brew.sh) for MacOS, or [Chocolatey](https://chocolatey.org) for Windows).

Especially on Windows, there are many ways of installing make (such as with [Windows Subsystem for Linux](https://docs.microsoft.com/en-us/windows/wsl/about), getting a binary from the [GNU Make for Windows](http://gnuwin32.sourceforge.net/packages/make.htm) page, etc.--I'm not going to cover them all here, but you can use what works best for you.

Likewise, this project is not opinionated about how you edit code. Use `vim` or `emacs`, [Atom](https://atom.io), [Sublime Text](https://www.sublimetext.com/), or a full IDE--whatever is most comfortable for you.

### Project structure

The application entrypoint is the `main` function in "main.go", in the root of the repository. This is the first thing that runs when you start the app.

To help understand the project layout further, or to modify it, check out the READMEs in [golang-standards/project-layout](https://github.com/golang-standards/project-layout).

### Versioning and release

The version number is currently determined by a variable in "version.sh". The `make app` command builds a binary that includes this version number (and also the name of the current git branch, if not "edge"). Remember to update this variable before tagging a commit as a release in GitHub, and also make sure the new tag matches the variable.

## Maintainers

- [Toma Morris](https://github.com/btmorr)

## Contributors

- [Toma Morris](https://github.com/btmorr)
- [Suhail Patel](https://github.com/suhailpatel)
- [Sameer Kolhar](https://github.com/kolharsam)
