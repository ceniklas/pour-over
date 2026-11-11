# pour-over

Type your password once, then let Homebrew run `brew update` and `brew upgrade` without stopping to ask for it again.

## How it works

Homebrew calls `sudo -A` when `SUDO_ASKPASS` is set. pour-over points `SUDO_ASKPASS` at its own binary, which in askpass mode just prints the password it was given. It also refreshes sudo's timestamp every 30 seconds so installers that call plain `sudo` stay authenticated too.

The password is read from the terminal without echo, checked with `sudo -v` up front, and passed to child processes through an environment variable. It is never written to disk.

## Install

```sh
go install github.com/ceniklas/pour-over@latest
```

Or build it yourself:

```sh
go build -o pour-over .
```

## Usage

```sh
pour-over                  # brew update && brew upgrade --no-ask
pour-over upgrade --greedy # run a single custom brew command instead
```

## Limitations

- Needs a terminal (it reads the password interactively) and an account with admin rights.
- Anything that asks for a password some way other than `sudo` (for example a GUI installer prompt) can't be answered.
