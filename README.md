# pour-over

<p align="center"><img src="pour-over.png" alt="Beer being poured through a pour-over coffee filter" width="420"><br><sub>Image generated with AI (ChatGPT).</sub></p>

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

## Security

You are handing this tool your sudo password, so here is exactly what happens to it:

- It is read from the terminal without echo and checked with `sudo -v` before anything else runs.
- It is held in memory only, never written to disk, never put in an environment variable and never passed on a command line.
- It is served over a Unix socket in a private temp directory (`0700`, socket `0600`) that exists only while pour-over runs.
- **Any process running as your user (or root) during that window could read it from the socket.** Homebrew's own installers run as you, so they can too. That is inherent to answering `sudo` prompts unattended; if you don't trust everything `brew upgrade` is about to run, don't use pour-over.

## Limitations

- Needs a terminal (it reads the password interactively) and an account with admin rights.
- Anything that asks for a password some way other than `sudo` (for example a GUI installer prompt) can't be answered.

## License

[MIT](LICENSE)

Co-Authored-By: AI-slop
