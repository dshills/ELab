# Run ELab in your browser

ELab runs a small program on your computer. It serves the editor and runs the
simulations while you work in your usual browser. Projects stay on this computer.
You do not need an account or an internet connection after downloading ELab.

## Start ELab

For a release with standalone downloads, choose the ZIP that matches your
computer in [GitHub Releases](https://github.com/dshills/ELab/releases). The
standalone ZIP contains the ELab program, the browser interface and
`START-HERE.txt`. The [published build 315](current-release.md) predates the
standalone downloads; its existing Apple Silicon app is still available. To use
the current source today, follow **Build from source** below.

1. Extract the ZIP and open a terminal in the extracted folder.
2. Run `./elab` on Linux or macOS, or `.\elab.exe` in Windows PowerShell.
3. Open [localhost:8788](http://localhost:8788) in your browser. If ELab reports
   that the port is in use, close the earlier ELab process or start the host
   with `--auto-port` and open the address it prints.

Leave the terminal open while you work. Press **Ctrl+C** in the terminal to save
and stop ELab. Closing the browser tab leaves the host running. On a new computer
or before upgrading, use **Projects → Back up library** to make a copy of your
projects and recovery history.

## Build from source

Install the Go version in [go.mod](../go.mod). From the project folder run
`go run .`, then open [localhost:8788](http://localhost:8788). To build a
standalone program, run `go build -o elab .` on Linux or macOS, or
`go build -o elab.exe .` on Windows. The program embeds the browser interface.
Opening `static/index.html` directly does not start the simulation engine.

## Optional Mac launcher

**ELab Web.app** starts the same local host and opens your default browser.
Its window shows the host status and has a **Quit** button that saves and stops
ELab. It uses the same library as the standalone program. Quit the older version
before switching between the two.

To build the launcher from source, use Go and the Xcode command-line tools:

```sh
python3 -B scripts/package_web.py --output 'build/local/ELab Web.app' \
  --version 0.209.0-dev --build-version 316 \
  --archive build/local/ELab-Web-dev-macos-launcher.zip
```

The optional bundle has a separate [signing and installation process](macos-distribution.md).
