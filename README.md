# pour-over

Do you install everything through Homebrew and casks? Then you know the enterprise annoyance: run `brew upgrade` and it asks for your password again and again, until you start thinking about weakening your password policy just to make it stop.

Don't. **pour-over** asks for your password once, checks it, and answers every `sudo` prompt Homebrew throws at it for the rest of the run: `brew update`, then `brew upgrade`, including the cask uninstalls and installer packages that normally interrupt you. Go get a coffee.

## How it works

Homebrew calls `sudo -A` when `SUDO_ASKPASS` is set. pour-over points `SUDO_ASKPASS` at a small wrapper script that runs pour-over itself in askpass mode. That helper reads the password from a private Unix socket (in a `0700` temp dir) served by the main process. It doesn't use environment variables, because Homebrew strips them for some steps such as cask uninstalls. pour-over also refreshes sudo's timestamp every 30 seconds so installers that call plain `sudo` stay authenticated.

The password is read from the terminal without echo, checked with `sudo -v` up front, and held only in memory. It is never written to disk, and the socket and temp dir are removed on exit.

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
