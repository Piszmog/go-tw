# Go Tool for Tailwind CSS

> [!NOTE]
> This project is not officially associated with or endorsed by Tailwind Labs Inc.

`go-tw` is an exectable to make it easier to integrate the Tailwindcss CLI into a 
Go project using the `go tool` directive.

Instead of having to have a separate install for `tailwindcss`, either with `npm` or 
the executable, a Go project can install and use `tailwindcss` with the `go tool` directive.

## Install

Add `go-tw` as a Go Tool by running the following command

```shell
go get -tool github.com/Piszmog/go-tw
```

## Run

Run `go-tw` as if it was the `tailwindcss` command. All arguments are piped to the
`tailwindcss` executable.

```shell
go-tw -i ./styles/input.css -o ./dist/assets/css/output@dev.css
```

### Tailwindcss Executable

When `go-tw` runs, it will install `tailwindcss` to your cache, for example `~/Library/Caches/go-tw` on macos.

By default, `go-tw` will check if a newer version of `tailwindcss` exists. If it does, it will download it and delete the older versions.

To use a specific version, provide the `-version` flag.

```shell
 ❯  go-tw -h -version v4.0.7
≈ tailwindcss v4.0.7

Usage:
  tailwindcss [--input input.css] [--output output.css] [--watch] [options…]

Options:
  -i, --input ··········· Input file
  -o, --output ·········· Output file [default: `-`]
  -w, --watch ··········· Watch for changes and rebuild as needed
  -m, --minify ·········· Optimize and minify the output
      --optimize ········ Optimize the output without minifying
      --cwd ············· The current working directory [default: `.`]
  -h, --help ············ Display usage information`
```

## Logging

`go-tw` has debug logging to help troubleshoot problems. Set the environment variable
`LOG_LEVEL` to `debug` to see the debug logs.

```shell
LOG_LEVEL=debug go-tw -h
```

